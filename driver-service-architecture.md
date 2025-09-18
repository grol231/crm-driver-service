# Driver Service - UML Диаграммы архитектуры

## Диаграмма компонентов сервиса

```mermaid
graph TB
    subgraph "Client Layer"
        WEB[Web Interface]
        MOBILE[Mobile App]
        ADMIN[Admin Panel]
    end
    
    subgraph "API Layer"
        REST[REST API<br/>Port: 8001]
        GRPC[gRPC API<br/>Port: 8001]
        WS[WebSocket<br/>Real-time tracking]
    end
    
    subgraph "Service Layer"
        DS[Driver Service]
        LS[Location Service]
        SS[Shift Service]
        RS[Rating Service]
        VS[Verification Service]
    end
    
    subgraph "Repository Layer"
        DR[Driver Repository]
        LR[Location Repository]
        SR[Shift Repository]
        RR[Rating Repository]
        DCR[Document Repository]
    end
    
    subgraph "Infrastructure Layer"
        PG[(PostgreSQL<br/>Primary DB)]
        REDIS[(Redis<br/>Cache & Sessions)]
        NATS[NATS<br/>Message Broker]
        S3[S3<br/>File Storage]
    end
    
    subgraph "External Services"
        GIBDD[GIBDD API<br/>Document Verification]
        MAPS[Maps API<br/>Geocoding]
        SMS[SMS Service]
    end
    
    %% Client connections
    WEB --> REST
    MOBILE --> REST
    MOBILE --> WS
    ADMIN --> REST
    ADMIN --> GRPC
    
    %% API to Services
    REST --> DS
    REST --> LS
    REST --> SS
    REST --> RS
    GRPC --> DS
    GRPC --> LS
    WS --> LS
    
    %% Service Layer connections
    DS --> VS
    DS --> DR
    DS --> DCR
    LS --> LR
    SS --> SR
    RS --> RR
    VS --> DCR
    
    %% Repository to Infrastructure
    DR --> PG
    LR --> PG
    SR --> PG
    RR --> PG
    DCR --> PG
    
    DS --> REDIS
    LS --> REDIS
    
    %% Message Broker
    DS --> NATS
    LS --> NATS
    SS --> NATS
    RS --> NATS
    
    %% File Storage
    DCR --> S3
    
    %% External integrations
    VS --> GIBDD
    LS --> MAPS
    DS --> SMS
    
    classDef client fill:#E8F4FD,stroke:#4A90E2,stroke-width:2px
    classDef api fill:#FF9500,stroke:#E6850E,stroke-width:2px,color:#fff
    classDef service fill:#4A90E2,stroke:#2E5BBA,stroke-width:2px,color:#fff
    classDef repository fill:#7ED321,stroke:#5BA517,stroke-width:2px
    classDef infrastructure fill:#9013FE,stroke:#6200EA,stroke-width:2px,color:#fff
    classDef external fill:#F5F5F5,stroke:#999,stroke-width:2px
    
    class WEB,MOBILE,ADMIN client
    class REST,GRPC,WS api
    class DS,LS,SS,RS,VS service
    class DR,LR,SR,RR,DCR repository
    class PG,REDIS,NATS,S3 infrastructure
    class GIBDD,MAPS,SMS external
```

## Диаграмма последовательности регистрации водителя

```mermaid
sequenceDiagram
    participant Client as Mobile App
    participant API as REST API
    participant DS as Driver Service
    participant VS as Verification Service
    participant DR as Driver Repository
    participant NATS as Message Broker
    participant GIBDD as GIBDD API
    participant S3 as File Storage
    
    Client->>+API: POST /api/v1/drivers
    API->>+DS: CreateDriver(driverData)
    
    DS->>+DR: CheckDriverExists(phone, license)
    DR-->>-DS: false
    
    DS->>+DR: CreateDriver(driver)
    DR->>PG: INSERT driver
    PG-->>DR: driver_id
    DR-->>-DS: driver
    
    DS->>+NATS: Publish "driver.registered"
    NATS-->>-DS: ack
    
    DS-->>-API: driver
    API-->>-Client: 201 Created
    
    Note over Client: User uploads documents
    
    Client->>+API: POST /drivers/{id}/documents
    API->>+DS: UploadDocument(driverID, file)
    
    DS->>+S3: UploadFile(file)
    S3-->>-DS: fileURL
    
    DS->>+DR: SaveDocument(driverID, docData)
    DR->>PG: INSERT document
    DR-->>-DS: document
    
    DS->>+VS: VerifyDocument(document)
    VS->>+GIBDD: CheckDocument(docNumber)
    GIBDD-->>-VS: isValid
    
    alt Document is valid
        VS-->>DS: verified
        DS->>+DR: UpdateDocumentStatus(docID, "verified")
        DR-->>-DS: success
        DS->>+NATS: Publish "driver.verified"
        NATS-->>-DS: ack
    else Document is invalid
        VS-->>DS: rejected
        DS->>+DR: UpdateDocumentStatus(docID, "rejected")
        DR-->>-DS: success
        DS->>+NATS: Publish "driver.document.rejected"
        NATS-->>-DS: ack
    end
    
    DS-->>-API: document
    API-->>-Client: 200 OK
```

## Диаграмма состояний водителя

```mermaid
stateDiagram-v2
    [*] --> Registered : Initial registration
    
    Registered --> PendingVerification : Documents uploaded
    PendingVerification --> Verified : All documents verified
    PendingVerification --> Rejected : Documents rejected
    PendingVerification --> Registered : Missing documents
    
    Rejected --> PendingVerification : Re-upload documents
    
    Verified --> Available : Ready to work
    Verified --> Suspended : Administrative action
    
    Available --> OnShift : Start shift
    Available --> Inactive : Manual status change
    Available --> Suspended : Policy violation
    
    OnShift --> Busy : Order assigned
    OnShift --> Available : End shift
    OnShift --> Inactive : Connection lost
    
    Busy --> OnShift : Order completed
    Busy --> Available : Order cancelled + End shift
    
    Inactive --> Available : Reconnected/Manual activation
    Inactive --> Suspended : Long inactivity
    
    Suspended --> Available : Suspension lifted
    Suspended --> Blocked : Serious violation
    
    Blocked --> [*] : Account terminated
    
    note right of Registered
        Driver can update profile
        and upload documents
    end note
    
    note right of OnShift
        GPS tracking active
        Can receive orders
    end note
    
    note right of Busy
        Order in progress
        Location tracked
    end note
```

## Диаграмма классов основных сущностей

```mermaid
classDiagram
    class Driver {
        +UUID id
        +String phone
        +String email
        +String firstName
        +String lastName
        +String middleName
        +Date birthDate
        +String passportSeries
        +String passportNumber
        +String licenseNumber
        +Date licenseExpiry
        +Status status
        +Float currentRating
        +Int totalTrips
        +Metadata metadata
        +DateTime createdAt
        +DateTime updatedAt
        +DateTime deletedAt
        
        +isActive() Bool
        +canReceiveOrders() Bool
        +updateRating(newRating Float)
        +incrementTripCount()
    }
    
    class DriverDocument {
        +UUID id
        +UUID driverId
        +DocumentType type
        +String documentNumber
        +Date issueDate
        +Date expiryDate
        +String fileURL
        +VerificationStatus status
        +String verifiedBy
        +DateTime verifiedAt
        +Metadata metadata
        
        +isExpired() Bool
        +isVerified() Bool
        +verify(verifierID String)
        +reject(reason String)
    }
    
    class DriverLocation {
        +UUID id
        +UUID driverId
        +Float latitude
        +Float longitude
        +Float altitude
        +Float accuracy
        +Float speed
        +Float bearing
        +String address
        +DateTime recordedAt
        
        +distanceTo(other DriverLocation) Float
        +isValidLocation() Bool
        +reverseGeocode() String
    }
    
    class DriverShift {
        +UUID id
        +UUID driverId
        +UUID vehicleId
        +DateTime startTime
        +DateTime endTime
        +ShiftStatus status
        +Location startLocation
        +Location endLocation
        +Int totalTrips
        +Float totalDistance
        +Float totalEarnings
        
        +getDuration() Duration
        +isActive() Bool
        +end(location DriverLocation)
        +addTrip(distance Float, earnings Float)
    }
    
    class DriverRating {
        +UUID id
        +UUID driverId
        +UUID orderId
        +UUID customerId
        +Int rating
        +String comment
        +RatingType type
        +Map criteriaScores
        +Bool isVerified
        
        +isValid() Bool
        +getOverallScore() Float
    }
    
    class RatingStats {
        +UUID driverId
        +Float averageRating
        +Int totalRatings
        +Map criteriaAverages
        +DateTime lastUpdated
        
        +calculate(ratings []DriverRating)
        +updateAverage(newRating Int)
    }
    
    %% Relationships
    Driver ||--o{ DriverDocument : has
    Driver ||--o{ DriverLocation : tracks
    Driver ||--o{ DriverShift : works
    Driver ||--o{ DriverRating : receives
    Driver ||--|| RatingStats : has
    
    DriverShift ||--o{ DriverLocation : contains
    DriverRating }o--|| DriverShift : belongs_to
```

## Диаграмма потоков данных

```mermaid
flowchart LR
    subgraph "Input Sources"
        MA[Mobile App]
        GPS[GPS Device]
        ADM[Admin Panel]
        EXT[External APIs]
    end
    
    subgraph "Driver Service Processing"
        VAL[Validation Layer]
        BL[Business Logic]
        CACHE[Cache Layer]
        EVT[Event Publisher]
    end
    
    subgraph "Data Storage"
        PG[(PostgreSQL)]
        RD[(Redis)]
        S3[(File Storage)]
    end
    
    subgraph "Output Streams"
        NATS[Message Broker]
        WS[WebSocket]
        API[REST API]
        RPT[Reports]
    end
    
    %% Input flows
    MA -->|Driver Registration<br/>Location Updates<br/>Shift Management| VAL
    GPS -->|Real-time Location<br/>Telematics Data| VAL
    ADM -->|Admin Actions<br/>Verifications| VAL
    EXT -->|Document Verification<br/>External Ratings| VAL
    
    %% Processing flows
    VAL -->|Validated Data| BL
    BL -->|Business Rules<br/>Applied| CACHE
    BL -->|Calculated Metrics<br/>State Changes| EVT
    
    %% Storage flows
    BL --> PG
    CACHE --> RD
    BL --> S3
    
    %% Output flows
    EVT -->|Events| NATS
    BL -->|Real-time Updates| WS
    BL -->|API Responses| API
    PG -->|Analytics Data| RPT
    
    %% Feedback loops
    NATS -.->|Event Responses| BL
    CACHE -.->|Cached Data| BL
    
    classDef input fill:#E8F4FD,stroke:#4A90E2,stroke-width:2px
    classDef process fill:#4A90E2,stroke:#2E5BBA,stroke-width:2px,color:#fff
    classDef storage fill:#9013FE,stroke:#6200EA,stroke-width:2px,color:#fff
    classDef output fill:#7ED321,stroke:#5BA517,stroke-width:2px
    
    class MA,GPS,ADM,EXT input
    class VAL,BL,CACHE,EVT process
    class PG,RD,S3 storage
    class NATS,WS,API,RPT output
```

## Диаграмма развертывания

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Nginx/HAProxy<br/>SSL Termination]
    end
    
    subgraph "Kubernetes Cluster"
        subgraph "Driver Service Pods"
            DS1[Driver Service<br/>Pod 1]
            DS2[Driver Service<br/>Pod 2]
            DS3[Driver Service<br/>Pod 3]
        end
        
        subgraph "Infrastructure Services"
            PG[PostgreSQL<br/>Master/Slave]
            RD[Redis Cluster<br/>3 nodes]
            NATS[NATS Cluster<br/>3 nodes]
        end
        
        subgraph "Monitoring"
            PROM[Prometheus]
            GRAF[Grafana]
            ALERT[AlertManager]
        end
    end
    
    subgraph "External Storage"
        S3[AWS S3<br/>Document Storage]
        CDN[CloudFront CDN]
    end
    
    subgraph "External APIs"
        GIBDD[GIBDD API]
        MAPS[Yandex Maps API]
        SMS[SMS Gateway]
    end
    
    %% Client connections
    CLIENT[Client Apps] --> LB
    LB --> DS1
    LB --> DS2
    LB --> DS3
    
    %% Service to infrastructure
    DS1 --> PG
    DS2 --> PG
    DS3 --> PG
    
    DS1 --> RD
    DS2 --> RD
    DS3 --> RD
    
    DS1 --> NATS
    DS2 --> NATS
    DS3 --> NATS
    
    %% External connections
    DS1 --> S3
    DS2 --> S3
    DS3 --> S3
    
    S3 --> CDN
    
    DS1 --> GIBDD
    DS2 --> GIBDD
    DS3 --> GIBDD
    
    DS1 --> MAPS
    DS2 --> MAPS
    DS3 --> MAPS
    
    DS1 --> SMS
    DS2 --> SMS
    DS3 --> SMS
    
    %% Monitoring connections
    DS1 --> PROM
    DS2 --> PROM
    DS3 --> PROM
    
    PROM --> GRAF
    PROM --> ALERT
    
    classDef service fill:#4A90E2,stroke:#2E5BBA,stroke-width:2px,color:#fff
    classDef infrastructure fill:#9013FE,stroke:#6200EA,stroke-width:2px,color:#fff
    classDef external fill:#F5F5F5,stroke:#999,stroke-width:2px
    classDef monitoring fill:#FF9500,stroke:#E6850E,stroke-width:2px,color:#fff
    classDef client fill:#E8F4FD,stroke:#4A90E2,stroke-width:2px
    
    class DS1,DS2,DS3 service
    class PG,RD,NATS infrastructure
    class S3,CDN,GIBDD,MAPS,SMS external
    class PROM,GRAF,ALERT monitoring
    class CLIENT,LB client
```

<<<<<<< HEAD
Эти UML диаграммы показывают полную архитектуру Driver Service, включая компоненты, потоки данных, состояния и развертывание в production среде.
=======
Эти UML диаграммы показывают полную архитектуру Driver Service, включая компоненты, потоки данных, состояния и развертывание в production среде.
>>>>>>> ac8533d8e091c50114bff809a58122508470f0f1
