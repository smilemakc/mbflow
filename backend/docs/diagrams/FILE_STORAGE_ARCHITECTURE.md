# File Storage Architecture Diagrams

## System Architecture

```mermaid
graph TB
    subgraph "Frontend Layer"
        UI[ResourcesPage UI]
        API_CLIENT[resourcesApi Service]
    end

    subgraph "REST API Layer"
        FSH[FileStorageHandlers]
        AUTH[AuthMiddleware]
    end

    subgraph "Application Layer"
        RFS[ResourceFileService]
        SM[StorageManager]
        PROV[LocalProvider]
    end

    subgraph "Infrastructure Layer"
        RR[ResourceRepository]
        FR[FileRepository]
        DB[(PostgreSQL)]
        FS[File System]
    end

    UI -->|HTTP + JWT| API_CLIENT
    API_CLIENT -->|REST API| FSH
    FSH -->|RequireAuth| AUTH
    FSH -->|Upload/Delete| RFS
    RFS -->|Check Quota| RR
    RFS -->|Store File| SM
    SM -->|Write| PROV
    PROV -->|Save| FS
    RFS -->|Save Metadata| FR
    FR -->|SQL| DB
    RR -->|SQL| DB

    style UI fill:#e1f5ff
    style FSH fill:#fff3e0
    style RFS fill:#f3e5f5
    style DB fill:#e8f5e9
    style FS fill:#fce4ec
```

## Upload File Flow

```mermaid
sequenceDiagram
    actor User
    participant UI as Frontend
    participant API as FileStorageHandlers
    participant Auth as AuthMiddleware
    participant Service as ResourceFileService
    participant Repo as ResourceRepository
    participant FileRepo as FileRepository
    participant Storage as StorageManager
    participant DB as Database

    User->>UI: Select & Upload File
    UI->>API: POST /resources/:id/files
    API->>Auth: Verify JWT Token
    Auth-->>API: User ID

    API->>Repo: GetByID(resourceID)
    Repo->>DB: SELECT resource
    DB-->>Repo: Resource
    Repo-->>API: FileStorageResource

    alt User owns resource
        API->>Service: UploadFile(resourceID, file)

        Service->>Repo: GetFileStorage(resourceID)
        Repo-->>Service: FileStorageResource

        Service->>Service: Check MaxFileSize
        Service->>Service: Check CanAddFile(size)

        alt Quota OK
            Service->>DB: BEGIN TRANSACTION

            Service->>Storage: Store(fileEntry, reader)
            Storage-->>Service: StoredFile

            Service->>FileRepo: Create(fileModel)
            FileRepo->>DB: INSERT INTO files

            Service->>Repo: IncrementUsage(resourceID, size)
            Repo->>DB: UPDATE resource_file_storage

            Service->>DB: COMMIT
            Service-->>API: FileModel

            API-->>UI: 201 Created + FileMetadata
            UI-->>User: Success Toast
        else Quota Exceeded
            Service-->>API: Error: Quota Exceeded
            API-->>UI: 507 Insufficient Storage
            UI-->>User: Error Toast
        end
    else User doesn't own resource
        API-->>UI: 403 Forbidden
        UI-->>User: Access Denied
    end
```

## Download File Flow

```mermaid
sequenceDiagram
    actor User
    participant UI as Frontend
    participant API as FileStorageHandlers
    participant Service as ResourceFileService
    participant FileRepo as FileRepository
    participant Storage as StorageManager
    participant FS as File System
    participant DB as Database

    User->>UI: Click Download
    UI->>API: GET /resources/:id/files/:file_id/download

    API->>Service: GetFile(resourceID, fileID)

    Service->>FileRepo: FindByID(fileID)
    FileRepo->>DB: SELECT FROM files
    DB-->>FileRepo: FileModel
    FileRepo-->>Service: FileModel

    Service->>Service: Verify resource_id matches
    Service->>Service: Check if expired

    alt File Valid
        Service->>Storage: Get(fileID)
        Storage->>FS: Read file
        FS-->>Storage: File Stream
        Storage-->>Service: Reader

        Service-->>API: FileModel + Reader
        API-->>UI: File Stream (200 OK)
        UI-->>User: Download File
    else File Expired
        Service-->>API: Error: Expired
        API-->>UI: 410 Gone
        UI-->>User: Error Toast
    end
```

## Delete File Flow

```mermaid
sequenceDiagram
    actor User
    participant UI as Frontend
    participant API as FileStorageHandlers
    participant Service as ResourceFileService
    participant FileRepo as FileRepository
    participant Repo as ResourceRepository
    participant Storage as StorageManager
    participant DB as Database

    User->>UI: Click Delete
    UI->>User: Confirm Dialog
    User->>UI: Confirm

    UI->>API: DELETE /resources/:id/files/:file_id

    API->>Service: DeleteFile(resourceID, fileID)

    Service->>FileRepo: FindByID(fileID)
    FileRepo-->>Service: FileModel

    Service->>Service: Verify ownership

    Service->>DB: BEGIN TRANSACTION

    Service->>Storage: Delete(fileID)
    Storage-->>Service: OK

    Service->>FileRepo: Delete(fileID)
    FileRepo->>DB: DELETE FROM files

    Service->>Repo: DecrementUsage(resourceID, size)
    Repo->>DB: UPDATE resource_file_storage

    Service->>DB: COMMIT
    Service-->>API: Success

    API-->>UI: 200 OK
    UI-->>User: Success Toast
```

## Quota Check Logic

```mermaid
flowchart TD
    A[Upload Request] --> B{File Size Valid?}
    B -->|No| C[Return 413 Too Large]
    B -->|Yes| D[Get Resource]

    D --> E[Load FileStorageResource]
    E --> F{Resource Active?}

    F -->|No| G[Return 400 Not Active]
    F -->|Yes| H{used + size <= limit?}

    H -->|No| I[Return 507 Quota Exceeded]
    H -->|Yes| J[Start Transaction]

    J --> K[Store File]
    K --> L[Save Metadata]
    L --> M[Increment Usage]
    M --> N{All OK?}

    N -->|Yes| O[Commit]
    N -->|No| P[Rollback]

    O --> Q[Return 201 Created]
    P --> R[Return 500 Error]

    style A fill:#e1f5ff
    style Q fill:#e8f5e9
    style C fill:#ffebee
    style G fill:#ffebee
    style I fill:#ffebee
    style R fill:#ffebee
```

## Database Schema Relations

```mermaid
erDiagram
    users ||--o{ resources : owns
    resources ||--|| resource_file_storage : "has details"
    resources ||--o{ files : contains
    pricing_plans ||--o{ resource_file_storage : "defines limits"

    users {
        uuid id PK
        string username
        string email
        string role
    }

    resources {
        uuid id PK
        uuid owner_id FK
        string type
        string name
        string status
        timestamp created_at
    }

    resource_file_storage {
        uuid resource_id PK,FK
        bigint storage_limit_bytes
        bigint used_storage_bytes
        int file_count
        uuid pricing_plan_id FK
    }

    files {
        uuid id PK
        uuid resource_id FK
        string name
        bigint size
        string mime_type
        string path
        timestamp created_at
    }

    pricing_plans {
        uuid id PK
        string name
        bigint storage_limit_bytes
        boolean is_free
    }
```

## Component Interaction

```mermaid
graph LR
    subgraph "Domain Layer"
        DR[FileStorageResource]
        DF[FileEntry]
    end

    subgraph "Application Layer"
        RFS[ResourceFileService]
        SM[StorageManager]
    end

    subgraph "Infrastructure"
        RR[ResourceRepository]
        FR[FileRepository]
        LP[LocalProvider]
    end

    RFS -->|Uses| DR
    RFS -->|Uses| DF
    RFS -->|Calls| RR
    RFS -->|Calls| FR
    RFS -->|Calls| SM

    SM -->|Uses| LP

    RR -->|Manages| DR
    FR -->|Manages| DF

    style DR fill:#f3e5f5
    style DF fill:#f3e5f5
    style RFS fill:#fff3e0
    style SM fill:#fff3e0
    style RR fill:#e8f5e9
    style FR fill:#e8f5e9
    style LP fill:#e8f5e9
```

## State Machine - File Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Uploading: User uploads file
    Uploading --> QuotaCheck: Validate

    QuotaCheck --> Active: Quota OK
    QuotaCheck --> Rejected: Quota exceeded

    Active --> Downloading: User downloads
    Downloading --> Active: Complete

    Active --> Expired: TTL reached
    Active --> Deleting: User deletes

    Expired --> Cleanup: Automatic
    Deleting --> Cleanup: Manual

    Cleanup --> [*]: Removed
    Rejected --> [*]: Never created

    note right of Active
        File stored and accessible
        Usage counters updated
    end note

    note right of QuotaCheck
        Check:
        - Max file size
        - Available space
        - Resource status
    end note
```

## Error Handling Flow

```mermaid
flowchart TD
    A[Request] --> B{Authenticated?}
    B -->|No| C[401 Unauthorized]
    B -->|Yes| D{Resource Exists?}

    D -->|No| E[404 Not Found]
    D -->|Yes| F{User Owns Resource?}

    F -->|No| G[403 Forbidden]
    F -->|Yes| H{Valid Operation?}

    H -->|Invalid Input| I[400 Bad Request]
    H -->|File Too Large| J[413 Entity Too Large]
    H -->|Quota Exceeded| K[507 Insufficient Storage]
    H -->|File Expired| L[410 Gone]
    H -->|Valid| M[Process Request]

    M --> N{Success?}
    N -->|Yes| O[200/201 Success]
    N -->|No| P[500 Internal Error]

    style O fill:#e8f5e9
    style C fill:#ffebee
    style E fill:#ffebee
    style G fill:#ffebee
    style I fill:#ffebee
    style J fill:#ffebee
    style K fill:#ffebee
    style L fill:#ffebee
    style P fill:#ffebee
```
