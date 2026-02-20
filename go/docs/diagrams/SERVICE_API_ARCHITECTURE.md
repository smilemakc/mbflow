# Service API - Architecture Diagrams

## System Architecture

```mermaid
graph TB
    subgraph "External Services"
        CRM[CRM Service]
        Billing[Billing Service]
        Analytics[Analytics Service]
        SDK[Go SDK ServiceClient]
    end

    subgraph "API Layer"
        AdminRoutes[Admin Routes<br>/service/system-keys]
        ServiceRoutes[Service API Routes<br>/api/v1/service/*]
        AuthMW[SystemAuthMiddleware]
        ImpersonationMW[HandleImpersonation]
        AuditMW[AuditMiddleware]
        JWTAuth[JWT AuthMiddleware]
    end

    subgraph "Application Layer"
        SystemKeySvc[SystemKey Service]
        AuditSvc[AuditService]
    end

    subgraph "Handlers"
        SysKeyH[SystemKeyHandlers]
        WorkflowH[WorkflowHandlers]
        ExecH[ExecutionHandlers]
        TriggerH[TriggerHandlers]
        CredH[CredentialHandlers]
        AuditH[AuditHandlers]
    end

    subgraph "Infrastructure Layer"
        SystemKeyRepo[SystemKeyRepository]
        AuditLogRepo[AuditLogRepository]
        WorkflowRepo[WorkflowRepository]
        ExecRepo[ExecutionRepository]
        TriggerRepo[TriggerRepository]
        CredRepo[CredentialsRepository]
        DB[(PostgreSQL)]
    end

    CRM -->|X-System-Key: sysk_...| ServiceRoutes
    Billing -->|X-System-Key: sysk_...| ServiceRoutes
    Analytics -->|X-System-Key: sysk_...| ServiceRoutes
    SDK -->|HTTP| ServiceRoutes

    AdminRoutes -->|JWT| JWTAuth
    JWTAuth --> SysKeyH
    SysKeyH --> SystemKeySvc

    ServiceRoutes --> AuthMW
    AuthMW --> ImpersonationMW
    ImpersonationMW --> AuditMW
    AuditMW --> WorkflowH
    AuditMW --> ExecH
    AuditMW --> TriggerH
    AuditMW --> CredH
    AuditMW --> AuditH

    AuthMW --> SystemKeySvc
    AuditMW --> AuditSvc

    SystemKeySvc --> SystemKeyRepo
    AuditSvc --> AuditLogRepo
    WorkflowH --> WorkflowRepo
    ExecH --> ExecRepo
    TriggerH --> TriggerRepo
    CredH --> CredRepo
    AuditH --> AuditSvc

    SystemKeyRepo --> DB
    AuditLogRepo --> DB
    WorkflowRepo --> DB
    ExecRepo --> DB
    TriggerRepo --> DB
    CredRepo --> DB

    style CRM fill:#e1f5ff
    style Billing fill:#e1f5ff
    style Analytics fill:#e1f5ff
    style SDK fill:#e1f5ff
    style AuthMW fill:#fff3e0
    style ImpersonationMW fill:#fff3e0
    style AuditMW fill:#fff3e0
    style SystemKeySvc fill:#f3e5f5
    style AuditSvc fill:#f3e5f5
    style DB fill:#e8f5e9
```

## Authentication Flow (JWT vs Service Key vs System Key)

```mermaid
flowchart TB
    Start([Client Request])
    Start --> CheckHeader{Authorization /<br>X-System-Key header?}

    CheckHeader -->|X-System-Key| ValidateSystemKey[Validate System Key]
    CheckHeader -->|Authorization: Bearer sk_...| ValidateServiceKey[Validate Service Key]
    CheckHeader -->|Authorization: Bearer jwt...| ValidateJWT[Validate JWT Token]
    CheckHeader -->|None| PublicCheck{Public endpoint?}

    PublicCheck -->|Yes| Allow[Allow Access]
    PublicCheck -->|No| Deny401[401 Unauthorized]

    ValidateSystemKey --> SysKeyValid{Valid?}
    SysKeyValid -->|Yes| CheckImpersonation{X-On-Behalf-Of<br>header?}
    SysKeyValid -->|No| Deny401

    CheckImpersonation -->|Yes| ValidateUser{User exists?}
    CheckImpersonation -->|No| UseSystemUser[Use System User ID]

    ValidateUser -->|Yes| SetImpersonated[Set impersonated user context]
    ValidateUser -->|No| Deny422[422 Invalid User]

    UseSystemUser --> RecordAudit[Record Audit Log]
    SetImpersonated --> RecordAudit

    ValidateServiceKey --> SKValid{Valid?}
    SKValid -->|Yes| SetUserCtx[Set user context]
    SKValid -->|No| Deny401

    ValidateJWT --> JWTValid{Valid?}
    JWTValid -->|Yes| SetUserCtx
    JWTValid -->|No| Deny401

    SetUserCtx --> Allow
    RecordAudit --> Allow

    Allow --> Handler([Route Handler])

    classDef success fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef error fill:#f44336,stroke:#c62828,color:#fff
    classDef check fill:#2196f3,stroke:#1565c0,color:#fff
    classDef process fill:#ff9800,stroke:#e65100,color:#fff

    class Allow,Handler success
    class Deny401,Deny422 error
    class CheckHeader,PublicCheck,SysKeyValid,CheckImpersonation,ValidateUser,SKValid,JWTValid check
    class ValidateSystemKey,ValidateServiceKey,ValidateJWT,UseSystemUser,SetImpersonated,SetUserCtx,RecordAudit process
```

## System Key Creation Flow

```mermaid
sequenceDiagram
    participant Admin
    participant REST as AdminHandlers
    participant JWT as JWT AuthMiddleware
    participant Service as SystemKey Service
    participant Crypto as crypto/rand + bcrypt
    participant Repo as SystemKeyRepository
    participant DB as PostgreSQL

    Admin->>REST: POST /api/v1/service/system-keys
    REST->>JWT: Validate JWT + RequireAdmin
    JWT-->>REST: Admin authenticated

    REST->>Service: CreateKey(name, serviceName, description, expiresInDays)

    Service->>Repo: Count()
    Repo->>DB: SELECT COUNT(*)
    DB-->>Repo: count
    Repo-->>Service: count

    alt count >= MaxKeys
        Service-->>REST: ErrSystemKeyLimitReached
        REST-->>Admin: 403 Forbidden
    else
        Service->>Crypto: Generate 32 random bytes
        Crypto-->>Service: random bytes

        Service->>Service: Base64URL encode + sysk_ prefix
        Service->>Crypto: bcrypt.GenerateFromPassword()
        Crypto-->>Service: keyHash

        Service->>Repo: Create(systemKey)
        Repo->>DB: INSERT INTO mbflow_system_keys
        DB-->>Repo: OK

        Service-->>REST: CreateResult{Key, PlainKey}
        REST-->>Admin: 201 Created

        Note over Admin: Plain key shown ONCE with warning
    end
```

## Service API Request Flow

```mermaid
sequenceDiagram
    participant Svc as External Service
    participant Auth as SystemAuthMiddleware
    participant Imp as HandleImpersonation
    participant Audit as AuditMiddleware
    participant Handler as Route Handler
    participant Repo as Repository
    participant AuditRepo as AuditLogRepo
    participant DB as PostgreSQL

    Svc->>Auth: GET /api/v1/service/workflows<br>X-System-Key: sysk_abc123...

    Auth->>Auth: Extract key from header
    Auth->>Auth: ValidateKey(plainKey)
    Note over Auth: prefix lookup → bcrypt verify → status check

    alt Key invalid
        Auth-->>Svc: 401 Unauthorized
    else Key valid
        Auth->>Auth: Set context: system_key_id, service_name
        Auth->>Imp: Next()

        Imp->>Imp: Check X-On-Behalf-Of header

        alt Has X-On-Behalf-Of
            Imp->>DB: SELECT user WHERE id = ?
            alt User exists
                Imp->>Imp: Set context: user_id, impersonated=true
            else User not found
                Imp-->>Svc: 422 User not found
            end
        else No impersonation
            Imp->>Imp: Set context: user_id=systemUserID
        end

        Imp->>Audit: Next()

        Audit->>Audit: Buffer request body

        Audit->>Handler: Next()

        Handler->>Repo: Execute operation
        Repo->>DB: SQL query
        DB-->>Repo: Result
        Repo-->>Handler: Data
        Handler-->>Audit: Response (status code)

        Audit->>Audit: parseServiceAPIPath(path, method)
        Note over Audit: /api/v1/service/workflows → action=workflow.list

        Audit-->>AuditRepo: Async: LogAction(...)
        AuditRepo->>DB: INSERT INTO mbflow_service_audit_log

        Audit-->>Svc: 200 OK + JSON response
    end
```

## Impersonation Model

```mermaid
flowchart TB
    Request([Service API Request]) --> HasHeader{X-On-Behalf-Of<br>header present?}

    HasHeader -->|Yes| ValidateUUID{Valid UUID format?}
    HasHeader -->|No| SystemMode[System Mode]

    ValidateUUID -->|No| Error422[422 Invalid Format]
    ValidateUUID -->|Yes| LookupUser[Lookup User in DB]

    LookupUser --> UserExists{User exists?}
    UserExists -->|No| Error422_2[422 User Not Found]
    UserExists -->|Yes| ImpersonatedMode[Impersonated Mode]

    SystemMode --> SetSystemUser["user_id = config.SystemUserID<br>impersonated = false"]
    ImpersonatedMode --> SetImpUser["user_id = X-On-Behalf-Of value<br>impersonated = true"]

    SetSystemUser --> Continue[Continue to handler]
    SetImpUser --> Continue

    Continue --> AuditLog["Audit log records:<br>- system_key_id (who)<br>- impersonated_user_id (on behalf of)<br>- action + resource"]

    style SystemMode fill:#e1f5ff
    style ImpersonatedMode fill:#fff3e0
    style Error422 fill:#ffebee
    style Error422_2 fill:#ffebee
    style AuditLog fill:#e8f5e9
```

## Audit Log Pipeline

```mermaid
sequenceDiagram
    participant MW as AuditMiddleware
    participant Parser as parseServiceAPIPath
    participant Sanitizer as sanitizeBody
    participant Service as AuditService
    participant Repo as AuditLogRepository
    participant DB as PostgreSQL

    MW->>MW: Read & buffer request body
    MW->>MW: Restore body for handler
    MW->>MW: c.Next() — execute handler

    Note over MW: After handler completes:

    MW->>MW: Get system_key_id from context
    MW->>MW: Get service_name from context
    MW->>MW: Get impersonated_user_id from context

    MW->>Parser: parseServiceAPIPath(path, method)
    Note over Parser: POST /api/v1/service/workflows<br>→ action="workflow.create"<br>→ resourceType="workflow"

    Note over Parser: GET /api/v1/service/workflows/:id<br>→ action="workflow.get"<br>→ resourceType="workflow"<br>→ resourceID=:id

    Note over Parser: POST /api/v1/service/executions/:id/cancel<br>→ action="execution.cancel"<br>→ resourceType="execution"<br>→ resourceID=:id

    MW-->>Service: go LogAction(...) — async goroutine

    Service->>Sanitizer: sanitizeBody(requestBody)
    Note over Sanitizer: Redacts fields containing:<br>key, secret, password,<br>token, credential, hash

    Sanitizer-->>Service: sanitized JSON

    Service->>Repo: Create(auditLogEntry)
    Repo->>DB: INSERT INTO mbflow_service_audit_log
```

## Database Schema

```mermaid
erDiagram
    USERS ||--o{ SYSTEM_KEYS : "created_by"
    SYSTEM_KEYS ||--o{ AUDIT_LOG : "authenticated_by"
    USERS ||--o{ AUDIT_LOG : "impersonated_as"

    SYSTEM_KEYS {
        uuid id PK
        varchar name
        text description
        varchar service_name
        varchar key_prefix "sysk_xxxxx"
        text key_hash "bcrypt"
        varchar status "active | revoked"
        timestamp last_used_at
        bigint usage_count
        timestamp expires_at
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
        timestamp revoked_at
    }

    AUDIT_LOG {
        uuid id PK
        uuid system_key_id FK
        varchar service_name
        uuid impersonated_user_id FK "nullable"
        varchar action "workflow.create"
        varchar resource_type "workflow"
        uuid resource_id "nullable"
        varchar request_method "POST"
        varchar request_path
        jsonb request_body "sanitized"
        int response_status
        varchar ip_address
        timestamp created_at
    }

    USERS {
        uuid id PK
        varchar email
        varchar username
        boolean is_active
        boolean is_admin
    }
```

## API Endpoints

```mermaid
graph LR
    subgraph "Admin Endpoints (JWT Auth)"
        A1["POST /service/system-keys"]
        A2["GET /service/system-keys"]
        A3["GET /service/system-keys/:id"]
        A4["DELETE /service/system-keys/:id"]
        A5["POST /service/system-keys/:id/revoke"]
    end

    subgraph "Service API (System Key Auth)"
        subgraph "Workflows"
            W1["GET /service/workflows"]
            W2["GET /service/workflows/:id"]
            W3["POST /service/workflows"]
            W4["PUT /service/workflows/:id"]
            W5["DELETE /service/workflows/:id"]
        end

        subgraph "Executions"
            E1["GET /service/executions"]
            E2["GET /service/executions/:id"]
            E3["POST /service/workflows/:id/execute"]
            E4["POST /service/executions/:id/cancel"]
            E5["POST /service/executions/:id/retry"]
        end

        subgraph "Triggers"
            T1["GET /service/triggers"]
            T2["POST /service/triggers"]
            T3["PUT /service/triggers/:id"]
            T4["DELETE /service/triggers/:id"]
        end

        subgraph "Credentials"
            C1["GET /service/credentials"]
            C2["POST /service/credentials"]
            C3["PUT /service/credentials/:id"]
            C4["DELETE /service/credentials/:id"]
        end

        subgraph "Audit"
            AU1["GET /service/audit-log"]
        end
    end

    style A1 fill:#fff3e0
    style A2 fill:#fff3e0
    style A3 fill:#fff3e0
    style A4 fill:#fff3e0
    style A5 fill:#fff3e0
```

## SDK ServiceClient Architecture

```mermaid
graph TB
    subgraph "User Code"
        App[External Service]
    end

    subgraph "sdk Package"
        SC[ServiceClient]
        WF[ServiceWorkflowsAPI]
        EX[ServiceExecutionsAPI]
        TR[ServiceTriggersAPI]
        CR[ServiceCredentialsAPI]
        DO[doRequest]
        DEC[decodeResponse]
    end

    subgraph "HTTP Layer"
        HC[http.Client]
    end

    subgraph "MBFlow Server"
        API[Service API Endpoints]
    end

    App -->|NewServiceClient| SC
    App -->|.As userID| SC

    SC --> WF
    SC --> EX
    SC --> TR
    SC --> CR

    WF --> DO
    EX --> DO
    TR --> DO
    CR --> DO

    DO -->|X-System-Key header<br>X-On-Behalf-Of header| HC
    HC -->|HTTPS| API

    API -->|JSON| DEC
    DEC --> WF
    DEC --> EX
    DEC --> TR
    DEC --> CR

    style SC fill:#f3e5f5
    style WF fill:#e1f5ff
    style EX fill:#e1f5ff
    style TR fill:#e1f5ff
    style CR fill:#e1f5ff
    style API fill:#e8f5e9
```

## System Key Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: Admin creates via API

    Created --> Active: Key stored with bcrypt hash

    Active --> Active: ValidateKey() success
    Active --> Active: usage_count++, last_used_at updated

    Active --> Revoked: RevokeKey()
    Active --> Expired: expires_at < NOW()

    Revoked --> [*]: DeleteKey()
    Expired --> [*]: DeleteKey()
    Active --> [*]: DeleteKey()

    note right of Created
        Plain key (sysk_...) shown
        to admin ONCE with warning.
        Only hash is stored.
    end note

    note right of Active
        Service authenticates with plain key.
        Prefix lookup → bcrypt verify.
        UpdateLastUsed on each use.
    end note

    note right of Revoked
        Cannot authenticate.
        revoked_at timestamp set.
        Can still be viewed/deleted.
    end note

    note right of Expired
        Cannot authenticate.
        Checked during validation.
        Automatic, no background job.
    end note
```

## Audit Log Body Sanitization

```mermaid
flowchart TD
    A[Request Body JSON] --> B{Parse as JSON?}
    B -->|No| C[Store as-is]
    B -->|Yes| D[Walk JSON keys recursively]

    D --> E{Key contains sensitive word?}

    E -->|Yes| F["Replace value with [REDACTED]"]
    E -->|No| G{Value is nested object?}

    G -->|Yes| D
    G -->|No| H[Keep value]

    F --> I[Continue to next key]
    H --> I
    I --> D

    D -->|All keys processed| J[Serialize to JSON]
    J --> K[Store sanitized body in audit log]

    subgraph "Sensitive Field Keywords"
        S1[key]
        S2[secret]
        S3[password]
        S4[token]
        S5[credential]
        S6[hash]
    end

    style F fill:#ffebee
    style K fill:#e8f5e9
    style S1 fill:#fff3e0
    style S2 fill:#fff3e0
    style S3 fill:#fff3e0
    style S4 fill:#fff3e0
    style S5 fill:#fff3e0
    style S6 fill:#fff3e0
```

## Middleware Chain

```mermaid
graph LR
    REQ([Request]) --> RSA[RequireSystemAccess]
    RSA --> HI[HandleImpersonation]
    HI --> RA[RecordAction]
    RA --> Handler[Route Handler]
    Handler --> RA
    RA --> RESP([Response])

    RSA -.->|Set| CTX1["ctx: system_key_id<br>ctx: service_name"]
    HI -.->|Set| CTX2["ctx: user_id<br>ctx: impersonated"]
    RA -.->|Async| AUDIT["Audit Log Entry<br>(goroutine)"]

    style RSA fill:#fff3e0
    style HI fill:#e1f5ff
    style RA fill:#f3e5f5
    style Handler fill:#e8f5e9
    style AUDIT fill:#fce4ec
```

## Error Handling

```mermaid
flowchart TD
    A[Service API Request] --> B{X-System-Key present?}
    B -->|No| C[401 Missing system key]
    B -->|Yes| D{Key starts with sysk_?}

    D -->|No| E[401 Invalid key format]
    D -->|Yes| F{Key found by prefix?}

    F -->|No| G[401 Invalid key]
    F -->|Yes| H{bcrypt matches?}

    H -->|No| G
    H -->|Yes| I{Key status active?}

    I -->|No| J[401 Key revoked]
    I -->|Yes| K{Key expired?}

    K -->|Yes| L[401 Key expired]
    K -->|No| M{X-On-Behalf-Of valid?}

    M -->|Invalid UUID| N[422 Invalid user ID]
    M -->|User not found| O[422 User not found]
    M -->|Valid or absent| P{Route exists?}

    P -->|No| Q[404 Not Found]
    P -->|Yes| R{Resource exists?}

    R -->|No| S[404 Not Found]
    R -->|Yes| T[200/201 Success]

    style T fill:#e8f5e9
    style C fill:#ffebee
    style E fill:#ffebee
    style G fill:#ffebee
    style J fill:#ffebee
    style L fill:#ffebee
    style N fill:#ffebee
    style O fill:#ffebee
    style Q fill:#ffebee
    style S fill:#ffebee
```

## Component Diagram

```mermaid
graph TB
    subgraph "systemkey Package"
        SVC[Service]
        AUDIT[AuditService]
        CFG[Config]
    end

    subgraph "models Package"
        SystemKey[SystemKey]
        AuditLog[ServiceAuditLog]
        Errors[Error Types]
    end

    subgraph "repository Package"
        SysKeyIface[SystemKeyRepository]
        AuditIface[ServiceAuditLogRepository]
        SysKeyFilter[SystemKeyFilter]
        AuditFilter[ServiceAuditLogFilter]
    end

    subgraph "storage Package"
        SysKeyImpl[SystemKeyRepoImpl]
        AuditImpl[ServiceAuditLogRepoImpl]
        SysKeyModel[SystemKeyModel - Bun]
        AuditModel[AuditLogModel - Bun]
        DB[(PostgreSQL)]
    end

    subgraph "rest Package"
        SysAuthMW[SystemAuthMiddleware]
        AuditMW[AuditMiddleware]
        SysKeyHandlers[SystemKeyHandlers]
        ServiceHandlers[6 Service Handlers]
    end

    subgraph "sdk Package"
        ServiceClient[ServiceClient]
        APIs[4 API Modules]
    end

    SVC --> CFG
    SVC --> SysKeyIface
    SVC --> SystemKey
    SVC --> Errors
    AUDIT --> AuditIface
    AUDIT --> AuditLog

    SysKeyIface --> SysKeyImpl
    AuditIface --> AuditImpl
    SysKeyImpl --> SysKeyModel
    AuditImpl --> AuditModel
    SysKeyImpl --> DB
    AuditImpl --> DB

    SysAuthMW --> SVC
    AuditMW --> AUDIT
    SysKeyHandlers --> SVC
    ServiceHandlers --> AuditMW

    ServiceClient --> APIs

    classDef service fill:#f3e5f5,stroke:#4a148c
    classDef model fill:#e1f5ff,stroke:#01579b
    classDef repo fill:#fff3e0,stroke:#e65100
    classDef infra fill:#e8f5e9,stroke:#1b5e20
    classDef handler fill:#fce4ec,stroke:#880e4f
    classDef sdk fill:#f1f8e9,stroke:#33691e

    class SVC,AUDIT,CFG service
    class SystemKey,AuditLog,Errors model
    class SysKeyIface,AuditIface,SysKeyFilter,AuditFilter repo
    class SysKeyImpl,AuditImpl,SysKeyModel,AuditModel,DB infra
    class SysAuthMW,AuditMW,SysKeyHandlers,ServiceHandlers handler
    class ServiceClient,APIs sdk
```

## Audit Log Retention

```mermaid
flowchart LR
    subgraph "Write Path"
        MW[AuditMiddleware] -->|Async goroutine| SVC[AuditService.LogAction]
        SVC --> REPO[Repository.Create]
        REPO --> DB[(audit_log table)]
    end

    subgraph "Read Path"
        API[GET /audit-log] --> LIST[AuditService.ListLogs]
        LIST --> FIND[Repository.FindAll]
        FIND --> DB
    end

    subgraph "Cleanup"
        CRON[Scheduled Job] --> CLEAN[AuditService.Cleanup]
        CLEAN --> DEL["Repository.DeleteOlderThan<br>(retention: 90 days)"]
        DEL --> DB
    end

    style MW fill:#fff3e0
    style DB fill:#e8f5e9
    style CRON fill:#fce4ec
```

## Notes

- All diagrams use Mermaid syntax for rendering in GitHub/IDE
- System keys (`sysk_` prefix) are separate from service keys (`sk_` prefix)
- System keys provide full superadmin access, service keys provide user-scoped access
- Audit log body is sanitized before storage to prevent credential leakage
- Audit logging is async (goroutine) to avoid impacting request latency
- Impersonation allows services to act on behalf of specific users
- SDK provides `.As(userID)` for client-level and `OnBehalfOf(userID)` for per-call impersonation
