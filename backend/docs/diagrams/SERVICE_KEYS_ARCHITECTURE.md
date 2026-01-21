# Service Keys - Architecture Diagrams

## System Architecture

```mermaid
graph TB
    subgraph "Client Layer"
        CLI[CLI Tool]
        Web[Web App]
        API[API Client]
    end

    subgraph "API Layer"
        REST[REST Handlers]
        Middleware[Auth Middleware]
    end

    subgraph "Application Layer"
        Service[Service Key Service]
        Config[Configuration]
    end

    subgraph "Domain Layer"
        Repository[Repository Interface]
        Models[Domain Models]
    end

    subgraph "Infrastructure Layer"
        Storage[PostgreSQL]
        Cache[Redis - Optional]
    end

    CLI -->|HTTP| REST
    Web -->|HTTP| REST
    API -->|HTTP| REST

    REST --> Middleware
    Middleware --> Service
    Service --> Repository
    Service --> Config
    Repository --> Storage
    Repository -.->|Optional| Cache

    Models -.->|Used by| Service
    Models -.->|Used by| Repository

    classDef client fill:#e1f5ff,stroke:#01579b,stroke-width:2px
    classDef api fill:#f3e5f5,stroke:#4a148c,stroke-width:2px
    classDef app fill:#fff3e0,stroke:#e65100,stroke-width:2px
    classDef domain fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px
    classDef infra fill:#fce4ec,stroke:#880e4f,stroke-width:2px

    class CLI,Web,API client
    class REST,Middleware api
    class Service,Config app
    class Repository,Models domain
    class Storage,Cache infra
```

## Key Creation Flow

```mermaid
sequenceDiagram
    participant Client
    participant REST
    participant Middleware
    participant Service
    participant Crypto
    participant Repository
    participant DB

    Client->>REST: POST /api/v1/service-keys
    REST->>Middleware: Validate JWT
    Middleware-->>REST: User authenticated

    REST->>Service: CreateKey(userID, name, description)

    Service->>Repository: CountByUserID(userID)
    Repository->>DB: SELECT COUNT(*)
    DB-->>Repository: count
    Repository-->>Service: count

    alt count >= MaxKeysPerUser
        Service-->>REST: ErrServiceKeyLimitReached
        REST-->>Client: 403 Forbidden
    else
        Service->>Crypto: generatePlainKey()
        Crypto-->>Service: plainKey (sk_xxx...)

        Service->>Crypto: hashKey(plainKey)
        Crypto-->>Service: keyHash (bcrypt)

        Service->>Repository: Create(serviceKey)
        Repository->>DB: INSERT INTO service_keys
        DB-->>Repository: OK
        Repository-->>Service: OK

        Service-->>REST: CreateResult{Key, PlainKey}
        REST-->>Client: 201 Created {key, plain_key}

        Note over Client: Plain key shown ONLY ONCE
    end
```

## Key Validation Flow

```mermaid
sequenceDiagram
    participant Client
    participant REST
    participant Middleware
    participant Service
    participant Repository
    participant DB

    Client->>REST: GET /api/v1/workflows
    Note over Client: Authorization: Bearer sk_a1b2c3...

    REST->>Middleware: Check Authorization
    Middleware->>Middleware: Detect service key (sk_ prefix)

    Middleware->>Service: ValidateKey(plainKey)

    Service->>Service: Extract prefix (first 8 chars)

    Service->>Repository: FindByPrefix(prefix)
    Repository->>DB: SELECT * WHERE key_prefix = ?
    DB-->>Repository: keys[]
    Repository-->>Service: keys[]

    loop For each key candidate
        Service->>Service: bcrypt.CompareHashAndPassword()

        alt Hash matches
            Service->>Service: CanUse() - check status & expiry

            alt Key is valid
                Service->>Repository: UpdateLastUsed(keyID)
                Repository->>DB: UPDATE service_keys SET last_used_at=NOW()

                Service-->>Middleware: ServiceKey{UserID, ...}
                Middleware->>Middleware: Set user context
                Middleware-->>REST: Continue request
                REST-->>Client: 200 OK + workflow data
            else Key revoked or expired
                Service-->>Middleware: Error (revoked/expired)
                Middleware-->>Client: 401 Unauthorized
            end
        end
    end

    alt No match found
        Service-->>Middleware: ErrInvalidServiceKey
        Middleware-->>Client: 401 Unauthorized
    end
```

## Authentication Flow (JWT vs Service Key)

```mermaid
flowchart TB
    Start([Client Request])
    Start --> CheckAuth{Authorization header?}

    CheckAuth -->|No| PublicEndpoint{Public endpoint?}
    CheckAuth -->|Yes| ParseHeader[Parse Authorization header]

    PublicEndpoint -->|Yes| AllowAccess[Allow access]
    PublicEndpoint -->|No| Deny401[401 Unauthorized]

    ParseHeader --> CheckPrefix{Starts with 'sk_'?}

    CheckPrefix -->|Yes| ValidateServiceKey[Validate Service Key]
    CheckPrefix -->|No| ValidateJWT[Validate JWT Token]

    ValidateServiceKey --> SKValid{Valid?}
    ValidateJWT --> JWTValid{Valid?}

    SKValid -->|Yes| SetUserContext[Set user context]
    SKValid -->|No| Deny401

    JWTValid -->|Yes| SetUserContext
    JWTValid -->|No| Deny401

    SetUserContext --> CheckPermissions{Has permission?}

    CheckPermissions -->|Yes| AllowAccess
    CheckPermissions -->|No| Deny403[403 Forbidden]

    AllowAccess --> Success([Continue to handler])
    Deny401 --> End([Return error])
    Deny403 --> End

    classDef success fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef error fill:#f44336,stroke:#c62828,color:#fff
    classDef check fill:#2196f3,stroke:#1565c0,color:#fff
    classDef process fill:#ff9800,stroke:#e65100,color:#fff

    class AllowAccess,Success success
    class Deny401,Deny403,End error
    class CheckAuth,PublicEndpoint,CheckPrefix,SKValid,JWTValid,CheckPermissions check
    class ParseHeader,ValidateServiceKey,ValidateJWT,SetUserContext process
```

## Data Flow

```mermaid
flowchart LR
    subgraph "Creation"
        C1[User Input] --> C2[Generate Random Bytes]
        C2 --> C3[Base64 Encode]
        C3 --> C4[Add sk_ Prefix]
        C4 --> C5[Bcrypt Hash]
        C5 --> C6[Store Hash + Prefix]
    end

    subgraph "Validation"
        V1[Submitted Key] --> V2[Extract Prefix]
        V2 --> V3[Query by Prefix]
        V3 --> V4[Get Candidates]
        V4 --> V5[Verify Bcrypt]
        V5 --> V6[Check Status]
        V6 --> V7[Update Usage]
    end

    subgraph "Storage"
        S1[(Database)]
        S2[key_prefix: sk_a1b2c]
        S3[key_hash: $2a$10$...]
        S4[status: active]
        S5[last_used_at]
        S6[usage_count]
    end

    C6 -.-> S1
    V3 -.-> S1
    V7 -.-> S1

    S1 --> S2
    S1 --> S3
    S1 --> S4
    S1 --> S5
    S1 --> S6

    classDef creation fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef validation fill:#2196f3,stroke:#1565c0,color:#fff
    classDef storage fill:#ff9800,stroke:#e65100,color:#fff

    class C1,C2,C3,C4,C5,C6 creation
    class V1,V2,V3,V4,V5,V6,V7 validation
    class S1,S2,S3,S4,S5,S6 storage
```

## Component Diagram

```mermaid
graph TB
    subgraph "servicekey Package"
        Service[Service]
        Config[Config]
        CreateResult[CreateResult]
    end

    subgraph "models Package"
        ServiceKey[ServiceKey]
        Errors[Error Types]
    end

    subgraph "repository Package"
        Interface[ServiceKeyRepository]
        Filter[ServiceKeyFilter]
    end

    subgraph "storage Package"
        Implementation[ServiceKeyRepositoryImpl]
        Database[(PostgreSQL)]
    end

    subgraph "crypto Package"
        Random[crypto/rand]
        BCrypt[golang.org/x/crypto/bcrypt]
    end

    Service --> Config
    Service --> CreateResult
    Service --> ServiceKey
    Service --> Errors
    Service --> Interface

    Interface <|.. Implementation
    Implementation --> Database
    Implementation --> Filter

    Service --> Random
    Service --> BCrypt

    classDef service fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef domain fill:#2196f3,stroke:#1565c0,color:#fff
    classDef repo fill:#ff9800,stroke:#e65100,color:#fff
    classDef infra fill:#9c27b0,stroke:#6a1b9a,color:#fff
    classDef crypto fill:#f44336,stroke:#c62828,color:#fff

    class Service,Config,CreateResult service
    class ServiceKey,Errors domain
    class Interface,Filter repo
    class Implementation,Database infra
    class Random,BCrypt crypto
```

## State Diagram

```mermaid
stateDiagram-v2
    [*] --> Created: CreateKey()

    Created --> Active: Validation successful
    Active --> Active: ValidateKey() (valid)

    Active --> Revoked: RevokeKey()
    Active --> Expired: Expiration date passed

    Revoked --> [*]: DeleteKey()
    Expired --> [*]: DeleteKey()

    Active --> [*]: DeleteKey()

    note right of Created
        Plain key shown to user
        Hash stored in database
    end note

    note right of Active
        Can be used for authentication
        last_used_at tracked
        usage_count incremented
    end note

    note right of Revoked
        Cannot be used
        Status: revoked
        revoked_at timestamp set
    end note

    note right of Expired
        Cannot be used
        expires_at < now
    end note
```

## Database Schema

```mermaid
erDiagram
    USERS ||--o{ SERVICE_KEYS : owns
    USERS ||--o{ SERVICE_KEYS : creates

    SERVICE_KEYS {
        uuid id PK
        uuid user_id FK
        varchar name
        text description
        varchar key_prefix
        text key_hash
        varchar status
        timestamp last_used_at
        bigint usage_count
        timestamp expires_at
        uuid created_by FK
        timestamp created_at
        timestamp updated_at
        timestamp revoked_at
    }

    USERS {
        uuid id PK
        varchar email
        varchar username
        text password_hash
        boolean is_active
        boolean is_admin
    }
```

## Security Model

```mermaid
flowchart TB
    subgraph "Key Generation"
        G1[32 random bytes] --> G2[Base64 URL encode]
        G2 --> G3[Add sk_ prefix]
        G3 --> G4{Unique?}
        G4 -->|No| G1
        G4 -->|Yes| G5[Bcrypt hash]
        G5 --> G6[Store hash + prefix]
    end

    subgraph "Key Storage"
        S1[Plain Key: NEVER stored]
        S2[Prefix: sk_a1b2c - Indexed]
        S3[Hash: $2a$10$... - Bcrypt]
        S4[Status: active/revoked]
        S5[Expiry: timestamp or null]
    end

    subgraph "Key Validation"
        V1[Extract prefix] --> V2[Query by prefix]
        V2 --> V3[Get candidates]
        V3 --> V4[Bcrypt verify]
        V4 --> V5{Match?}
        V5 -->|No| V6[Try next]
        V5 -->|Yes| V7[Check status]
        V7 --> V8{Active?}
        V8 -->|Yes| V9[Check expiry]
        V8 -->|No| V10[Deny]
        V9 --> V11{Expired?}
        V11 -->|No| V12[Allow]
        V11 -->|Yes| V10
    end

    G6 -.-> S2
    G6 -.-> S3
    S2 -.-> V2

    classDef gen fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef store fill:#2196f3,stroke:#1565c0,color:#fff
    classDef validate fill:#ff9800,stroke:#e65100,color:#fff
    classDef decision fill:#f44336,stroke:#c62828,color:#fff

    class G1,G2,G3,G5,G6 gen
    class S1,S2,S3,S4,S5 store
    class V1,V2,V3,V4,V6,V7,V9,V12 validate
    class G4,V5,V8,V11,V10 decision
```

## Performance Characteristics

```mermaid
graph LR
    subgraph "Create Key"
        C1[Count check: 5ms]
        C2[Generate key: 1ms]
        C3[Bcrypt hash: 100ms]
        C4[DB insert: 5ms]
        C5[Total: ~111ms]
    end

    subgraph "Validate Key"
        V1[Prefix query: 5ms]
        V2[Bcrypt verify: 100ms]
        V3[Update usage: 3ms]
        V4[Total: ~108ms]
    end

    subgraph "List Keys"
        L1[DB query: 5ms]
        L2[Total: ~5ms]
    end

    subgraph "Revoke Key"
        R1[Find key: 3ms]
        R2[Update status: 3ms]
        R3[Total: ~6ms]
    end

    classDef fast fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef medium fill:#ff9800,stroke:#e65100,color:#fff
    classDef slow fill:#f44336,stroke:#c62828,color:#fff

    class L1,L2,R1,R2,R3,C1,C2,C4,V1,V3 fast
    class C5,V4 medium
    class C3,V2 slow
```

## Deployment Architecture

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[NGINX / ALB]
    end

    subgraph "Application Servers"
        API1[MBFlow API 1]
        API2[MBFlow API 2]
        API3[MBFlow API 3]
    end

    subgraph "Data Layer"
        PG[(PostgreSQL Primary)]
        PGR[(PostgreSQL Replica)]
        Redis[(Redis - Optional)]
    end

    Client[Client] --> LB
    LB --> API1
    LB --> API2
    LB --> API3

    API1 --> PG
    API2 --> PG
    API3 --> PG

    API1 -.->|Read-only| PGR
    API2 -.->|Read-only| PGR
    API3 -.->|Read-only| PGR

    API1 -.->|Cache| Redis
    API2 -.->|Cache| Redis
    API3 -.->|Cache| Redis

    PG -.->|Replication| PGR

    classDef lb fill:#4caf50,stroke:#2e7d32,color:#fff
    classDef api fill:#2196f3,stroke:#1565c0,color:#fff
    classDef data fill:#ff9800,stroke:#e65100,color:#fff

    class LB lb
    class API1,API2,API3 api
    class PG,PGR,Redis data
```

## Notes

- All diagrams use Mermaid syntax for easy rendering
- Security is enforced at multiple layers
- Performance optimized with indexed prefix lookups
- Stateless validation allows horizontal scaling
- Optional Redis caching can reduce DB load
