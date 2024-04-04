create table presto_benchmarks.presto_query_creation_info
(
    query_id                varchar(255) not null comment 'A unique identifier for every Presto query. The query_id can be used across different presto tables to join together information about the query',
    query                   text         null comment 'The query text of the Presto query. This is a protected column and can be accessed for allowed queries using the view `presto_query_statistics_view`',
    create_time             timestamp    null comment 'Unix timestamp for the time when the server received the query',
    schema_name             varchar(255) null comment 'For most of the queries this just points to the namespace name',
    catalog_name            varchar(255) null comment 'If you are looking for warehouse queries, then consider using `prism` for interactive queries(Daiquery, Unidash) and `prism_batch` for batch queries (Dataswarm)',
    environment             varchar(255) null comment 'Name of the Presto cluster on which this query was executed',
    user                    varchar(255) null comment 'Name of the user',
    remote_client_address   varchar(255) null comment 'IP address of the machine that initiated the Presto request',
    source                  varchar(255) null comment 'An identifier for every Presto query. For dataswarm users, this normally is of the form `dataswarm@<usecase, example: worker, tester>@<dataswarm_task_id, example: di.presto.generate_queries>`, for API users, this just contains the source as passed by the user',
    user_agent              varchar(255) null comment 'Information about the client making the request to Presto',
    uri                     varchar(255) null comment 'This URL is meant to get information about the Presto query - that said - it is very short lived URL and would not be very useful to most users',
    session_properties_json longtext     null comment 'List of all the session properties passed along with the values',
    server_version          varchar(255) null comment 'The Presto version that was deployed on the cluster when this query was executed',
    client_info             varchar(255) null comment 'Usually a JSON (but it is not enforced so ensure that your queries consider that) and contains detailed information about query context. For people looking for chronos job instance information you can access the JSON PATH `$.source_info.chronos_jiid`',
    resource_group_name     varchar(255) null comment 'Resource groups are Presto''s way of distributing resources to different customers. This field is the resource group in which the user''s group got executed. Documentation - https://www.internalfb.com/intern/wiki/Presto/how-do-resource-groups-work/',
    principal               varchar(255) null comment 'The TLS credential that is used when lauching the query on Presto',
    transaction_id          varchar(255) null,
    client_tags             text         null comment 'The clients tags are tags that are manually added by customers and also automatically added by Presto. These tags help with selecting various execution parameters like resource groups and other settings',
    resource_estimates      varchar(255) null comment 'These are estimates about query execution metrics which can either be passed by a user manually or in certain situations predicted by the user. These estimates help in finding the best resource group to execute your query to optimize for latency and reliability',
    dt                      datetime(3)  not null,
    primary key (query_id, dt)
)
    partition by key (`dt`) partitions 365;

create table presto_benchmarks.presto_query_operator_stats
(
    query_id                             varchar(255) not null,
    stage_id                             bigint       not null,
    stage_execution_id                   bigint       null,
    pipeline_id                          bigint       not null,
    operator_id                          bigint       not null,
    plan_node_id                         varchar(255) null,
    operator_type                        varchar(255) null,
    total_drivers                        bigint       null,
    add_input_calls                      bigint       null,
    add_input_wall_ms                    bigint       null,
    add_input_cpu_ms                     bigint       null,
    add_input_allocation_bytes           bigint       null,
    raw_input_data_size_bytes            bigint       null,
    raw_input_positions                  bigint       null,
    input_data_size_bytes                bigint       null,
    input_positions                      bigint       null,
    sum_squared_input_positions          double       null,
    get_output_calls                     bigint       null,
    get_output_wall_ms                   bigint       null,
    get_output_cpu_ms                    bigint       null,
    get_output_allocation_bytes          bigint       null,
    output_data_size_bytes               bigint       null,
    output_positions                     bigint       null,
    physical_written_data_size_bytes     bigint       null,
    blocked_wall_ms                      bigint       null,
    finish_calls                         bigint       null,
    finish_wall_ms                       bigint       null,
    finish_cpu_ms                        bigint       null,
    finish_allocation_bytes              bigint       null,
    user_memory_reservation_bytes        bigint       null,
    revocable_memory_reservation_bytes   bigint       null,
    system_memory_reservation_bytes      bigint       null,
    peak_user_memory_reservation_bytes   bigint       null,
    peak_system_memory_reservation_bytes bigint       null,
    peak_total_memory_reservation_bytes  bigint       null,
    spilled_data_size_bytes              bigint       null,
    info                                 text         null,
    runtime_stats                        text         null,
    dt                                   datetime(3)  not null,
    primary key (query_id, stage_id, pipeline_id, operator_id, dt)
)
    partition by key (`dt`) partitions 365;

create table presto_benchmarks.presto_query_plans
(
    query_id    varchar(255) not null,
    query       longtext     null,
    plan        longtext     null,
    json_plan   longtext     null,
    environment varchar(255) null,
    dt          datetime(3)  not null,
    primary key (query_id, dt)
)
    partition by key (`dt`,`query_id`) partitions 365;

create table presto_benchmarks.presto_query_stage_stats
(
    query_id                         varchar(255) not null,
    stage_id                         bigint       not null,
    stage_execution_id               bigint       null,
    tasks                            bigint       null,
    total_scheduled_time_ms          bigint       null,
    total_cpu_time_ms                bigint       null,
    retried_cpu_time_ms              bigint       null,
    total_blocked_time_ms            bigint       null,
    raw_input_data_size_bytes        bigint       null,
    processed_input_data_size_bytes  bigint       null,
    physical_written_data_size_bytes bigint       null,
    gc_statistics                    text         null,
    cpu_distribution                 text         null,
    memory_distribution              text         null,
    dt                               datetime(3)  not null,
    primary key (query_id, stage_id, dt)
)
    partition by key (`dt`,`query_id`,`stage_id`) partitions 365;

create table presto_benchmarks.presto_query_statistics
(
    query_id                   varchar(255) not null comment 'A unique identifier for every Presto query. The query_id can be used across different presto tables to join together information about the query',
    query                      text         null comment 'The query text of the Presto query. This is a protected column and can be accessed for allowed queries using the view `presto_query_statistics_view`',
    query_type                 varchar(255) null comment 'Describes the type of query. This is a best effort and might not be filled for every query',
    schema_name                varchar(255) null comment 'For most of the queries this just points to the namespace name',
    catalog_name               varchar(255) null comment 'If you are looking for warehouse queries, then consider using `prism` for interactive queries(Daiquery, Unidash) and `prism_batch` for batch queries (Dataswarm)',
    environment                varchar(255) null comment 'Name of the Presto cluster on which this query was executed',
    user                       varchar(255) null comment 'Name of the user',
    remote_client_address      varchar(255) null comment 'IP address of the machine that initiated the Presto request',
    source                     varchar(255) null comment 'An identifier for every Presto query. For dataswarm users, this normally is of the form `dataswarm@<usecase, example: worker, tester>@<dataswarm_task_id, example: di.presto.generate_queries>`, for API users, this just contains the source as passed by the user',
    user_agent                 varchar(255) null comment 'Information about the client making the request to Presto',
    uri                        varchar(255) null comment 'This URL is meant to get information about the Presto query - that said - it is very short lived URL and would not be very useful to most users',
    session_properties_json    longtext     null comment 'List of all the session properties passed along with the values',
    server_version             varchar(255) null comment 'The Presto version that was deployed on the cluster when this query was executed',
    client_info                varchar(255) null comment 'Usually a JSON (but it is not enforced so ensure that your queries consider that) and contains detailed information about query context. For people looking for chronos job instance information you can access the JSON PATH `$.source_info.chronos_jiid`',
    resource_group_name        varchar(255) null comment 'Resource groups are Presto''s way of distributing resources to different customers. This field is the resource group in which the user''s group got executed. Documentation - https://www.internalfb.com/intern/wiki/Presto/how-do-resource-groups-work/',
    principal                  varchar(255) null comment 'The TLS credential that is used when lauching the query on Presto',
    transaction_id             varchar(255) null,
    client_tags                text         null comment 'The clients tags are tags that are manually added by customers and also automatically added by Presto. These tags help with selecting various execution parameters like resource groups and other settings',
    resource_estimates         varchar(255) null comment 'These are estimates about query execution metrics which can either be passed by a user manually or in certain situations predicted by the user. These estimates help in finding the best resource group to execute your query to optimize for latency and reliability',
    create_time                datetime     null comment 'Unix timestamp for the time when the server received the query',
    end_time                   datetime     null comment 'Unix timestamp for the time when the query finished',
    execution_start_time       datetime     null comment 'Unix timestamp for the time when the query exited the queue and started the actual execution',
    query_state                varchar(255) null comment 'Tells if the query finished successfully or failed',
    failure_message            longtext     null comment 'If the query failed, the failure message, otherwise null',
    failure_type               varchar(255) null comment 'The Java exception class, null if query did not fail',
    failures_json              longtext     null,
    failure_task               varchar(255) null comment 'In case the failure happened during query execution, the exact task which had the failure',
    failure_host               varchar(255) null comment 'The host on which the failure_task was running',
    error_code                 bigint       null comment 'A numeric code representing the error. `error_code_name` is a better field to use',
    error_code_name            varchar(255) null comment 'Error code corresponding to the error',
    error_category             varchar(255) null comment 'Consists of 4 categories - USER_ERROR(consists of errors like Syntax errors or incorrect table/column names), INSUFFICIENT_RESOURCES(Crossing memory limits, execution time limits), EXTERNAL(Errors caused by systems outside of Presto, like WarmStorage or Hive Metastore), INTERNAL(Errors caused by Presto itself, like node restarts)',
    warnings_json              longtext     null comment 'Warnings issued during the course of query execution. These warnings can be for various reasons - inefficient queries, use of deprecated features and more',
    splits                     bigint       null comment 'Number of Presto splits executed',
    analysis_time_ms           bigint       null comment 'Time taken to analyze the query',
    queued_time_ms             bigint       null comment 'Time that the query spent waiting inside the resource group queues',
    planning_time_ms           bigint       null comment 'Time that the query spent in planning phase',
    query_wall_time_ms         bigint       null comment 'Total wall time from start to end taken by a query (includes both queuing and execution)',
    query_execution_time_ms    bigint       null comment 'Total wall time that the query spent executing (does not include queue time)',
    bytes_per_cpu_sec          bigint       null comment 'Input bytes read per cpu second of computation',
    bytes_per_sec              bigint       null comment 'This is computed by dividing the total bytes read by the query by the total query execution time (excludes the queuing time)',
    rows_per_cpu_sec           bigint       null comment 'Input rows read per cpu second of computation',
    total_bytes                bigint       null comment 'Total input bytes read by the query',
    total_rows                 bigint       null comment 'Total input rows read by the query',
    output_rows                bigint       null comment 'Total rows returned back to the client.',
    output_bytes               bigint       null comment 'Total bytes returned back to the client.',
    written_rows               bigint       null comment 'Total rows written to an output table',
    written_bytes              bigint       null comment 'Total bytes written to an output table',
    cumulative_memory          double       null comment 'Total amount of user memory used by the query over the course of its execution. The unit is bytes-ms',
    peak_user_memory_bytes     bigint       null comment 'The peak memory consumed by the query for user related constructs (example: hash tables for join, aggregations) over the whole query execution',
    peak_total_memory_bytes    bigint       null comment 'The peak memory consumed by the query for both user related and internal constructs (example: data structures for scans and during table writes) over the whole query execution',
    peak_task_total_memory     bigint       null comment 'The peak total memory in bytes for any of the tasks',
    peak_task_user_memory      bigint       null comment 'The peak user memory in bytes for any of the tasks',
    written_intermediate_bytes bigint       null comment 'Bytes written to storage between stages',
    peak_node_total_memory     bigint       null comment 'The peak total memory used by a query on any host',
    total_split_cpu_time_ms    bigint       null comment 'The total CPU time consumed by a query',
    stage_count                bigint       null comment 'Total number of stages in a query',
    cumulative_total_memory    double       null comment 'Total amount of total memory used the query over the source of its execution. The unit is bytes-ms',
    dt                         datetime(3)  not null,
    primary key (dt, query_id)
)
    partition by key (`dt`) partitions 365;
