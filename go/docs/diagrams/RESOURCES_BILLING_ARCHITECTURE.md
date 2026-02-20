# Resources and Billing Architecture

## System Overview

```mermaid
graph TB
    subgraph "Client Layer"
        WebClient[Web Client]
        APIClient[API Client]
    end

    subgraph "API Layer"
        Router[Gin Router]
        AuthMW[Auth Middleware]
        ResourceHandler[Resource Handlers]
        AccountHandler[Account Handlers]
        FileHandler[File Handlers]
    end

    subgraph "Domain Layer"
        ResourceRepo[Resource Repository Interface]
        AccountRepo[Account Repository Interface]
        TxRepo[Transaction Repository Interface]
        PlanRepo[Pricing Plan Repository Interface]
    end

    subgraph "Infrastructure Layer"
        ResourceRepoImpl[Resource Repo Implementation]
        AccountRepoImpl[Account Repo Implementation]
        TxRepoImpl[Transaction Repo Implementation]
        PlanRepoImpl[Pricing Plan Repo Implementation]
    end

    subgraph "Data Layer"
        PostgreSQL[(PostgreSQL)]
    end

    subgraph "Domain Models"
        Resource[Resource Model]
        FileStorage[FileStorage Model]
        Account[Account Model]
        Transaction[Transaction Model]
        PricingPlan[Pricing Plan Model]
    end

    WebClient --> Router
    APIClient --> Router
    Router --> AuthMW
    AuthMW --> ResourceHandler
    AuthMW --> AccountHandler
    AuthMW --> FileHandler

    ResourceHandler --> ResourceRepo
    ResourceHandler --> PlanRepo
    AccountHandler --> AccountRepo
    AccountHandler --> TxRepo
    FileHandler --> ResourceRepo

    ResourceRepo --> ResourceRepoImpl
    AccountRepo --> AccountRepoImpl
    TxRepo --> TxRepoImpl
    PlanRepo --> PlanRepoImpl

    ResourceRepoImpl --> PostgreSQL
    AccountRepoImpl --> PostgreSQL
    TxRepoImpl --> PostgreSQL
    PlanRepoImpl --> PostgreSQL

    ResourceHandler --> Resource
    ResourceHandler --> FileStorage
    AccountHandler --> Account
    AccountHandler --> Transaction
    ResourceHandler --> PricingPlan

    style WebClient fill:#e1f5ff
    style APIClient fill:#e1f5ff
    style Router fill:#fff3e0
    style AuthMW fill:#f3e5f5
    style ResourceHandler fill:#e8f5e9
    style AccountHandler fill:#e8f5e9
    style FileHandler fill:#e8f5e9
    style PostgreSQL fill:#e0f2f1
```

## Request Flow

### Resource Creation Flow

```mermaid
sequenceDiagram
    participant Client
    participant AuthMW as Auth Middleware
    participant Handler as Resource Handler
    participant Repo as Repository
    participant DB as PostgreSQL
    participant Model as Domain Model

    Client->>AuthMW: POST /resources/file-storage + JWT
    AuthMW->>AuthMW: Validate Token
    AuthMW->>AuthMW: Extract User ID
    AuthMW->>Handler: Forward Request (with user_id)

    Handler->>Handler: Bind JSON & Validate
    Handler->>Model: NewFileStorageResource(userID, name)
    Model->>Model: Set Defaults (5MB limit)
    Model->>Handler: Return Resource

    Handler->>Repo: Create(resource)
    Repo->>DB: INSERT INTO resources
    Repo->>DB: INSERT INTO file_storage
    DB->>Repo: Success
    Repo->>Handler: Success

    Handler->>Client: 201 Created + Resource JSON
```

### Deposit Flow with Idempotency

```mermaid
sequenceDiagram
    participant Client
    participant Handler as Account Handler
    participant TxRepo as Transaction Repo
    participant AccRepo as Account Repo
    participant DB as PostgreSQL

    Client->>Handler: POST /account/deposit + idempotency_key
    Handler->>AccRepo: GetByUserID(userID)
    AccRepo->>DB: SELECT FROM accounts
    DB->>AccRepo: Account Data
    AccRepo->>Handler: Account

    Handler->>Handler: Check Account Status

    Handler->>TxRepo: GetByIdempotencyKey(key)
    alt Existing Transaction
        TxRepo->>DB: SELECT FROM transactions
        DB->>TxRepo: Existing Transaction
        TxRepo->>Handler: Transaction Found
        Handler->>Client: 200 OK + Existing Transaction (duplicate_request: true)
    else New Transaction
        TxRepo->>Handler: Not Found
        Handler->>Handler: account.Deposit(amount)
        Handler->>AccRepo: UpdateBalance(accountID, newBalance)
        AccRepo->>DB: UPDATE accounts SET balance
        DB->>AccRepo: Success

        Handler->>Handler: Create Transaction Object
        Handler->>TxRepo: Create(transaction)
        TxRepo->>DB: INSERT INTO transactions
        DB->>TxRepo: Success
        TxRepo->>Handler: Success

        Handler->>Client: 200 OK + New Transaction
    end
```

## Data Model

```mermaid
erDiagram
    USERS ||--o{ ACCOUNTS : has
    USERS ||--o{ RESOURCES : owns
    ACCOUNTS ||--o{ TRANSACTIONS : contains
    RESOURCES ||--|| FILE_STORAGE : is_a
    PRICING_PLANS ||--o{ FILE_STORAGE : applies_to

    USERS {
        uuid id PK
        string email
        string username
        timestamp created_at
    }

    ACCOUNTS {
        uuid id PK
        uuid user_id FK
        decimal balance
        string currency
        string status
        timestamp created_at
        timestamp updated_at
    }

    TRANSACTIONS {
        uuid id PK
        uuid account_id FK
        string type
        decimal amount
        string currency
        string status
        string idempotency_key UK
        decimal balance_before
        decimal balance_after
        timestamp created_at
    }

    RESOURCES {
        uuid id PK
        string type
        uuid owner_id FK
        string name
        string description
        string status
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }

    FILE_STORAGE {
        uuid resource_id PK,FK
        bigint storage_limit_bytes
        bigint used_storage_bytes
        int file_count
        uuid pricing_plan_id FK
    }

    PRICING_PLANS {
        uuid id PK
        string resource_type
        string name
        decimal price_per_unit
        string unit
        bigint storage_limit_bytes
        string billing_period
        string pricing_model
        boolean is_free
        boolean is_active
    }
```

## Component Diagram

```mermaid
graph LR
    subgraph "handlers_resources.go"
        CreateFS[CreateFileStorage]
        ListRes[ListResources]
        GetRes[GetResource]
        UpdateRes[UpdateResource]
        DeleteRes[DeleteResource]
        ListPlans[ListPricingPlans]
    end

    subgraph "handlers_accounts.go"
        GetAcc[GetAccount]
        Deposit[Deposit]
        ListTx[ListTransactions]
        GetTx[GetTransaction]
    end

    subgraph "middleware_auth.go"
        RequireAuth[RequireAuth]
        OptionalAuth[OptionalAuth]
        GetUserID[GetUserID]
    end

    subgraph "Repositories"
        FileStorageRepo[FileStorageRepository]
        AccountRepo[AccountRepository]
        TransactionRepo[TransactionRepository]
        PricingPlanRepo[PricingPlanRepository]
    end

    CreateFS --> RequireAuth
    CreateFS --> GetUserID
    CreateFS --> FileStorageRepo

    ListRes --> RequireAuth
    ListRes --> FileStorageRepo

    Deposit --> RequireAuth
    Deposit --> AccountRepo
    Deposit --> TransactionRepo

    ListPlans --> PricingPlanRepo

    style CreateFS fill:#e8f5e9
    style Deposit fill:#e8f5e9
    style RequireAuth fill:#f3e5f5
    style FileStorageRepo fill:#fce4ec
```

## State Machine: Account Status

```mermaid
stateDiagram-v2
    [*] --> Active: Create Account
    Active --> Suspended: Suspend()
    Suspended --> Active: Activate()
    Active --> Closed: Close()
    Suspended --> Closed: Close()
    Closed --> [*]

    note right of Active
        Can perform all operations
        Deposits and charges allowed
    end note

    note right of Suspended
        Read-only access
        No charges allowed
    end note

    note right of Closed
        No operations allowed
        Account archived
    end note
```

## State Machine: Transaction Status

```mermaid
stateDiagram-v2
    [*] --> Pending: Create Transaction
    Pending --> Completed: Complete()
    Pending --> Failed: Fail()
    Completed --> Reversed: Reverse()
    Failed --> [*]
    Reversed --> [*]
    Completed --> [*]

    note right of Completed
        Funds applied to balance
        Immutable state
    end note

    note right of Reversed
        Original transaction reversed
        New reverse transaction created
    end note
```

## Resource Usage Flow

```mermaid
graph TB
    Start[File Upload Request]
    CheckAuth{Authenticated?}
    GetResource[Get Resource]
    CheckQuota{Space Available?}
    StoreFile[Store File]
    UpdateMetrics[Update Usage Metrics]
    Success[Upload Success]
    Error1[Auth Error]
    Error2[Quota Exceeded]

    Start --> CheckAuth
    CheckAuth -->|No| Error1
    CheckAuth -->|Yes| GetResource
    GetResource --> CheckQuota
    CheckQuota -->|No| Error2
    CheckQuota -->|Yes| StoreFile
    StoreFile --> UpdateMetrics
    UpdateMetrics --> Success

    style Start fill:#e1f5ff
    style Success fill:#e8f5e9
    style Error1 fill:#ffebee
    style Error2 fill:#ffebee
    style CheckAuth fill:#f3e5f5
    style CheckQuota fill:#fff3e0
```

## API Endpoint Hierarchy

```mermaid
graph TD
    API["/api/v1"]

    Resources["/resources"]
    Account["/account"]
    Files["/files"]

    API --> Resources
    API --> Account
    API --> Files

    Resources --> CreateFS["POST /file-storage"]
    Resources --> ListRes["GET /"]
    Resources --> GetRes["GET /:id"]
    Resources --> UpdateRes["PUT /:id"]
    Resources --> DeleteRes["DELETE /:id"]
    Resources --> Plans["GET /pricing-plans"]

    Account --> GetAcc["GET /"]
    Account --> Deposit["POST /deposit"]
    Account --> Transactions["GET /transactions"]
    Account --> GetTx["GET /transactions/:id"]

    Files --> Upload["POST /"]
    Files --> ListFiles["GET /"]
    Files --> GetFile["GET /:id"]
    Files --> DeleteFile["DELETE /:id"]

    style API fill:#e1f5ff
    style Resources fill:#e8f5e9
    style Account fill:#fff3e0
    style Files fill:#f3e5f5
```

## Security Layers

```mermaid
graph TB
    Request[HTTP Request]

    subgraph "Security Layers"
        JWT[JWT Token Validation]
        UserExtract[Extract User ID]
        Ownership[Check Resource Ownership]
        Status[Check Account Status]
        Quota[Check Resource Quota]
    end

    Handler[Handler Logic]
    DB[(Database)]

    Request --> JWT
    JWT --> UserExtract
    UserExtract --> Ownership
    Ownership --> Status
    Status --> Quota
    Quota --> Handler
    Handler --> DB

    JWT -.->|401 Unauthorized| Error1[Reject]
    Ownership -.->|403 Forbidden| Error2[Reject]
    Status -.->|403 Forbidden| Error3[Reject]
    Quota -.->|429 Quota Exceeded| Error4[Reject]

    style JWT fill:#f3e5f5
    style Ownership fill:#fff3e0
    style Status fill:#fff3e0
    style Quota fill:#ffebee
    style Error1 fill:#ffebee
    style Error2 fill:#ffebee
    style Error3 fill:#ffebee
    style Error4 fill:#ffebee
```

## Performance Considerations

### Caching Strategy

```mermaid
graph LR
    Request[API Request]
    Cache{Cache Hit?}
    GetFromCache[Get from Cache]
    GetFromDB[Get from Database]
    UpdateCache[Update Cache]
    Response[Return Response]

    Request --> Cache
    Cache -->|Yes| GetFromCache
    Cache -->|No| GetFromDB
    GetFromCache --> Response
    GetFromDB --> UpdateCache
    UpdateCache --> Response

    style Cache fill:#fff3e0
    style GetFromCache fill:#e8f5e9
    style UpdateCache fill:#f3e5f5
```

**Cacheable Data:**
- Pricing Plans (rarely change)
- User Resources List (5-minute TTL)
- Account Balance (1-minute TTL)

**Non-Cacheable:**
- Transaction History (real-time data)
- Resource Usage Metrics (frequently updated)

## Error Handling Flow

```mermaid
graph TD
    Handler[Handler Receives Request]
    Validate{Validation OK?}
    Auth{Authorized?}
    Domain{Domain Logic OK?}
    DB{DB Operation OK?}
    Success[Return Success]

    Handler --> Validate
    Validate -->|No| BadRequest[400 Bad Request]
    Validate -->|Yes| Auth
    Auth -->|No| Unauthorized[401/403]
    Auth -->|Yes| Domain
    Domain -->|No| BusinessError[400/422]
    Domain -->|Yes| DB
    DB -->|No| ServerError[500]
    DB -->|Yes| Success

    style Success fill:#e8f5e9
    style BadRequest fill:#ffebee
    style Unauthorized fill:#ffebee
    style BusinessError fill:#fff3e0
    style ServerError fill:#ffebee
```

## Deployment Architecture

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Nginx/HAProxy]
    end

    subgraph "Application Tier"
        App1[MBFlow Server 1]
        App2[MBFlow Server 2]
        App3[MBFlow Server 3]
    end

    subgraph "Data Tier"
        PG[(PostgreSQL Primary)]
        PGR[(PostgreSQL Replica)]
        Redis[(Redis Cache)]
    end

    Client[Client] --> LB
    LB --> App1
    LB --> App2
    LB --> App3

    App1 --> PG
    App2 --> PG
    App3 --> PG

    App1 --> PGR
    App2 --> PGR
    App3 --> PGR

    App1 --> Redis
    App2 --> Redis
    App3 --> Redis

    PG -.->|Replication| PGR

    style LB fill:#e1f5ff
    style App1 fill:#e8f5e9
    style App2 fill:#e8f5e9
    style App3 fill:#e8f5e9
    style PG fill:#e0f2f1
    style PGR fill:#e0f2f1
    style Redis fill:#ffebee
```
