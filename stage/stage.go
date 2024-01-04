package stage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"presto-benchmark/log"
	"presto-benchmark/presto"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type GetClientFn func() *presto.Client
type OnQueryCompletionFn func(result *QueryResult)

var DefaultServerUrl = "http://127.0.0.1:8080"
var DefaultGetClientFn = func() *presto.Client {
	client, _ := presto.NewClient(DefaultServerUrl)
	return client
}

type Stage struct {
	// Id is used to uniquely identify a stage. It is usually the file name without its directory path and extension.
	Id string `json:"-"`
	// Catalog, Schema, and SessionParams will be inherited by the children stages unless a new client is created
	// by setting start_on_new_client = true on children stages.
	Catalog       *string        `json:"catalog,omitempty"`
	Schema        *string        `json:"schema,omitempty"`
	SessionParams map[string]any `json:"session_params,omitempty"`
	Queries       []string       `json:"queries,omitempty"`
	// If a stage has both Queries and QueryFiles, the queries in the Queries array will be executed first then
	// the QueryFiles will be read and executed.
	QueryFiles []string `json:"query_files,omitempty"`
	// If StartOnNewClient is set to true, this stage will create a new client to execute itself.\
	// This new client will be passed down to its descendant stages unless those stages also set StartOnNewClient to true.
	// Each client can carry their own set of client information, tags, session properties, user credentials, etc.
	// Descendant stages will **NOT** inherit this value from their predecessors.
	StartOnNewClient bool `json:"start_on_new_client,omitempty"`
	// If AbortOnError is set to true, the context associated with this stage will be canceled if an error occurs.
	// Depending on when the cancellable context was created, this may abort some or all other running stages and all future stages.
	// Children stages will inherit this value from their parent if it is not set.
	AbortOnError *bool `json:"abort_on_error,omitempty"`
	// If SaveOutput is set to true, the query result will be saved to files in its raw form.
	// Children stages will inherit this value from their parent if it is not set.
	SaveOutput *bool `json:"save_output,omitempty"`
	// If SaveJson is set to true, the query json will be saved to files in its raw form.
	// Children stages will inherit this value from their parent if it is not set.
	SaveJson       *bool    `json:"save_json,omitempty"`
	OutputPath     string   `json:"output_path,omitempty"`
	NextStagePaths []string `json:"next,omitempty"`
	BaseDir        string   `json:"-"`
	Prerequisites  []*Stage `json:"-"`
	NextStages     []*Stage `json:"-"`
	// Client is by default passed down to descendant stages.
	Client *presto.Client `json:"-"`
	// GetClient is called when the stage needs to create a new Presto client. This function is passed down to descendant stages by default.
	GetClient GetClientFn `json:"-"`
	// AbortAll is passed down to descendant stages by default and will be used to cancel the current context.
	AbortAll          context.CancelCauseFunc `json:"-"`
	OnQueryCompletion OnQueryCompletionFn     `json:"-"`
	// wgPrerequisites is a count-down latch to wait for all the prerequisites to finish before starting this stage.
	wgPrerequisites sync.WaitGroup
	// wgExitMainStage blocks the main stage from exiting before it counts down to zero, so we can wait for other
	// goroutines to finish.
	wgExitMainStage *sync.WaitGroup
	// started is used to make sure only one goroutine is started to run this stage when there are multiple prerequisites.
	started    atomic.Bool
	resultChan chan *QueryResult
}

func (s *Stage) waitForPrerequisites() <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		s.wgPrerequisites.Wait()
		close(ch)
	}()
	return ch
}

// Run this stage and trigger its downstream stages.
func (s *Stage) Run(ctx context.Context) []*QueryResult {
	results := make([]*QueryResult, 0, len(s.Queries)+len(s.QueryFiles))
	s.resultChan = make(chan *QueryResult)
	s.wgExitMainStage = &sync.WaitGroup{}
	s.wgExitMainStage.Add(1)

	// create output directory
	if s.OutputPath == "" {
		s.OutputPath = s.BaseDir
	}
	s.OutputPath = filepath.Join(s.OutputPath, fmt.Sprintf("%s-%s", s.Id, time.Now().Format(time.RFC3339)))
	if err := os.MkdirAll(s.OutputPath, 0755); err != nil {
		log.Fatal().Str("output_path", s.OutputPath).
			Err(err).Msg("failed to create output directory")
	} else {
		log.Info().Str("output_path", s.OutputPath).
			Msg("output will be saved in this path")
	}

	logPath := filepath.Join(s.OutputPath, s.Id+".log")
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644); err != nil {
		log.Error().Str("log_path", logPath).
			Err(err).Msg("failed to create log file")
	} else {
		bufWriter := bufio.NewWriter(logFile)
		defer bufWriter.Flush()
		log.SetGlobalLogger(zerolog.New(io.MultiWriter(os.Stdout, bufWriter)).With().Timestamp().Stack().Logger())
	}

	timeToExit := make(chan os.Signal, 1)
	signal.Notify(timeToExit, os.Interrupt, os.Kill)
	go func() {
		s.wgExitMainStage.Wait()
		close(timeToExit)
	}()

	ctx, s.AbortAll = context.WithCancelCause(ctx)
	log.Debug().EmbedObject(s).Msg("created cancellable context")
	go func() {
		_ = s.run(ctx)
	}()

	b := strings.Builder{}
	b.WriteString("stage_id,query_file,query_index,info_url,succeeded,row_count,start_time,end_time,duration_in_seconds\n")
	for {
		select {
		case result := <-s.resultChan:
			results = append(results, result)

			b.WriteString(result.StageId + ",")
			if result.QueryFile != nil {
				b.WriteString(*result.QueryFile)
			} else {
				b.WriteString("inline")
			}
			b.WriteString(fmt.Sprintf(",%d,%s,%t,%d,%s,%s,%f\n",
				result.QueryIndex, result.InfoUrl, result.QueryError == nil, result.RowCount,
				result.StartTime.Format(time.RFC3339), result.EndTime.Format(time.RFC3339), result.Duration.Seconds()))
		case sig := <-timeToExit:
			if sig != nil {
				s.AbortAll(fmt.Errorf(sig.String()))
			}
			_ = os.WriteFile(filepath.Join(s.OutputPath, "summary.csv"), []byte(b.String()), 0644)
			return results
		}
	}
}

func (s *Stage) run(ctx context.Context) (returnErr error) {
	if !s.started.CompareAndSwap(false, true) {
		// If other prerequisites finished earlier, then this stage is already called and waiting.
		s.wgExitMainStage.Done()
		return nil
	}
	defer func() {
		for _, nextStage := range s.NextStages {
			nextStage.wgPrerequisites.Done()
		}
		if returnErr != nil {
			s.logErr(ctx, returnErr)
			if *s.AbortOnError && s.AbortAll != nil {
				log.Debug().EmbedObject(s).Msg("canceling the context because abort_on_error is set to true")
				s.AbortAll(returnErr)
			}
		} else {
			// Trigger descendant stages.
			s.wgExitMainStage.Add(len(s.NextStages))
			for _, nextStage := range s.NextStages {
				go func(nextStage *Stage) {
					_ = nextStage.run(ctx)
				}(nextStage)
			}
		}
		s.wgExitMainStage.Done()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.waitForPrerequisites():
		log.Debug().EmbedObject(s).Msg("all prerequisites finished")
	}
	s.prepareClient()
	s.propagateStates()
	if err := s.runQueries(ctx, s.Queries, nil); err != nil {
		return err
	}
	for i := 0; i < len(s.QueryFiles); i++ {
		file, err := os.Open(s.QueryFiles[i])
		if err != nil {
			return err
		}
		queries, err := presto.SplitQueries(file)
		if err != nil {
			return err
		}
		err = s.runQueries(ctx, queries, &s.QueryFiles[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Stage) SaveQueryJsonFile(ctx context.Context, result *QueryResult) {
	if !*s.SaveJson && result.QueryError == nil {
		return
	}
	queryJsonFile, err := os.OpenFile(
		filepath.Join(s.OutputPath, querySource(s, result))+".json",
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err == nil {
		queryJson := bufio.NewWriterSize(queryJsonFile, 8192)
		_, err = s.Client.GetQueryInfo(ctx, result.QueryId, queryJson)
		if err == nil {
			err = queryJson.Flush()
			if err == nil {
				err = queryJsonFile.Close()
			}
		}
	}
	if err != nil {
		log.Error().Err(err).EmbedObject(result.SimpleLogging()).
			Msg("failed to write query json")
	}
}

func (s *Stage) runQuery(ctx context.Context, queryIndex int, query string, queryFile *string) (result *QueryResult, retErr error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().EmbedObject(s).Msgf("recovered from panic: %v", r)
			if e, ok := r.(error); ok {
				retErr = e
			}
		}
		result.QueryError = retErr
		result.ConcludeExecution()
	}()

	result = &QueryResult{
		StageId:    s.Id,
		Query:      query,
		QueryFile:  queryFile,
		QueryIndex: queryIndex,
		StartTime:  time.Now(),
	}

	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	querySourceStr := querySource(s, result)
	clientResult, _, err := s.Client.Query(ctx, query,
		func(req *http.Request) {
			req.Header.Set(presto.SourceHeader, querySourceStr)
		})
	if clientResult != nil {
		result.QueryId = clientResult.Id
		result.InfoUrl = clientResult.InfoUri
	}
	if err != nil {
		return result, err
	}

	// Log query submission
	e := log.Debug().EmbedObject(result.SimpleLogging())
	if catalog := s.Client.GetCatalog(); catalog != "" {
		e = e.Str("catalog", catalog)
	}
	if schema := s.Client.GetSchema(); schema != "" {
		e = e.Str("schema", schema)
	}
	if *s.SaveOutput {
		e.Bool("save_output", true)
	}
	e.Msgf("submitted query")

	var (
		queryOutputFile *os.File
		queryOutput     *bufio.Writer
		wgQueryOutput   *sync.WaitGroup
	)
	if *s.SaveOutput {
		queryOutputFile, err = os.OpenFile(
			filepath.Join(s.OutputPath, querySourceStr)+".output",
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return result, err
		}
		queryOutput = bufio.NewWriterSize(queryOutputFile, 8192)
		wgQueryOutput = &sync.WaitGroup{}
	}

	err = clientResult.Drain(ctx, func(qr *presto.QueryResults) error {
		result.RowCount += len(qr.Data)
		if queryOutput == nil {
			return nil
		}
		// Write output asynchronously. wgExitMainStage prevents the program from exiting if there is still
		// output data to write after all the queries finish running.
		s.wgExitMainStage.Add(1)
		// wgQueryOutput waits for all the writes to complete then flush and close the file.
		wgQueryOutput.Add(1)
		go func(data []json.RawMessage) {
			defer func() {
				wgQueryOutput.Done()
				s.wgExitMainStage.Done()
			}()
			for _, row := range data {
				_, ioErr := queryOutput.Write(row)
				if ioErr == nil {
					ioErr = queryOutput.WriteByte('\n')
				}
				if ioErr != nil {
					log.Error().Err(ioErr).EmbedObject(result.SimpleLogging()).
						Msg("failed to write query result")
					return
				}
			}
		}(qr.Data)
		return nil
	})

	// Now all the data for this query is being written
	if queryOutput != nil {
		s.wgExitMainStage.Add(1)
		go func() {
			wgQueryOutput.Wait()
			if ioErr := queryOutput.Flush(); ioErr != nil {
				log.Error().Err(ioErr).EmbedObject(result.SimpleLogging()).
					Msg("failed to write query result")
			} else {
				log.Debug().EmbedObject(result.SimpleLogging()).Msg("query data saved successfully")
			}
			_ = queryOutputFile.Close()
			s.wgExitMainStage.Done()
		}()
	}
	return result, err
}

func (s *Stage) runQueries(ctx context.Context, queries []string, queryFile *string) (retErr error) {
	for i, query := range queries {
		result, err := s.runQuery(ctx, i, query, queryFile)

		if s.OnQueryCompletion != nil {
			s.OnQueryCompletion(result)
		}
		s.SaveQueryJsonFile(ctx, result)
		// Each query should have a query result sent to the channel, no matter
		// its execution succeeded or not.
		s.resultChan <- result
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				// If the context is cancelled or timed out, we cannot continue whatsoever and must return.
				return result
			}
			if *s.AbortOnError {
				// Skip the rest queries in the same batch.
				// Logging etc. will be handled in the parent stack.
				return result
			}
			// Log the error information and continue running
			s.logErr(ctx, result)
			continue
		}
		log.Info().EmbedObject(result).Msgf("query finished")
	}
	return nil
}
