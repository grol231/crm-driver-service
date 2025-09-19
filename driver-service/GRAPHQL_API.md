# GraphQL API для Driver Service

Этот документ описывает GraphQL API для Driver Service, включая схему, запросы, мутации и примеры использования.

## Обзор

GraphQL API предоставляет гибкий интерфейс для взаимодействия с Driver Service, позволяя клиентам запрашивать только необходимые данные и выполнять операции над водителями, их местоположениями, рейтингами, сменами и документами.

## Endpoints

- **GraphQL endpoint**: `POST /graphql`
- **GraphQL Playground** (только для development): `GET /playground`

## Основные типы

### Driver (Водитель)
```graphql
type Driver {
  id: UUID!
  phone: String!
  email: String!
  firstName: String!
  lastName: String!
  middleName: String
  fullName: String!
  birthDate: Time!
  licenseNumber: String!
  licenseExpiry: Time!
  status: Status!
  currentRating: Float!
  totalTrips: Int!
  createdAt: Time!
  updatedAt: Time!
  
  # Связанные объекты
  documents: [DriverDocument!]!
  currentLocation: DriverLocation
  locationHistory(limit: Int, offset: Int): [DriverLocation!]!
  ratings(limit: Int, offset: Int): [DriverRating!]!
  ratingStats: RatingStats
  activeShift: DriverShift
  shifts(limit: Int, offset: Int): [DriverShift!]!
  
  # Вычисляемые поля
  isActive: Boolean!
  canReceiveOrders: Boolean!
  isLicenseExpired: Boolean!
}
```

### DriverLocation (Местоположение водителя)
```graphql
type DriverLocation {
  id: UUID!
  driverId: UUID!
  latitude: Float!
  longitude: Float!
  altitude: Float
  accuracy: Float
  speed: Float
  bearing: Float
  address: String
  recordedAt: Time!
  createdAt: Time!
  
  # Связанные объекты
  driver: Driver!
  
  # Вычисляемые поля
  isHighAccuracy: Boolean!
  isValidLocation: Boolean!
}
```

### DriverRating (Рейтинг водителя)
```graphql
type DriverRating {
  id: UUID!
  driverId: UUID!
  orderId: UUID
  customerId: UUID
  rating: Int!
  comment: String
  ratingType: RatingType!
  criteriaScores: CriteriaScores
  isVerified: Boolean!
  isAnonymous: Boolean!
  createdAt: Time!
  updatedAt: Time!
  
  # Связанные объекты
  driver: Driver!
  
  # Вычисляемые поля
  isValid: Boolean!
  overallScore: Float!
}
```

### DriverShift (Смена водителя)
```graphql
type DriverShift {
  id: UUID!
  driverId: UUID!
  vehicleId: UUID
  status: ShiftStatus!
  startTime: Time!
  endTime: Time
  startLocation: Location
  endLocation: Location
  totalTrips: Int!
  totalDistance: Float!
  totalEarnings: Float!
  fuelConsumed: Float
  createdAt: Time!
  updatedAt: Time!
  
  # Связанные объекты
  driver: Driver!
  
  # Вычисляемые поля
  duration: Int! # в минутах
  isActive: Boolean!
  averageEarningsPerTrip: Float!
  averageDistancePerTrip: Float!
  earningsPerHour: Float!
}
```

## Основные запросы (Queries)

### Получить водителя по ID
```graphql
query GetDriver($id: UUID!) {
  driver(id: $id) {
    id
    phone
    email
    firstName
    lastName
    fullName
    status
    currentRating
    totalTrips
    isActive
    canReceiveOrders
    isLicenseExpired
    currentLocation {
      latitude
      longitude
      address
      recordedAt
    }
    ratingStats {
      averageRating
      totalRatings
      percentile95
      percentile90
    }
  }
}
```

### Получить список водителей с фильтрами
```graphql
query GetDrivers($filters: DriverFilters, $limit: Int, $offset: Int) {
  drivers(filters: $filters, limit: $limit, offset: $offset) {
    drivers {
      id
      phone
      firstName
      lastName
      status
      currentRating
      totalTrips
      createdAt
    }
    pageInfo {
      total
      limit
      offset
      hasMore
    }
  }
}
```

### Получить активных водителей
```graphql
query GetActiveDrivers {
  activeDrivers {
    id
    phone
    firstName
    lastName
    status
    currentLocation {
      latitude
      longitude
    }
  }
}
```

### Найти ближайших водителей
```graphql
query GetNearbyDrivers($latitude: Float!, $longitude: Float!, $radiusKm: Float, $limit: Int) {
  nearbyDrivers(latitude: $latitude, longitude: $longitude, radiusKm: $radiusKm, limit: $limit) {
    id
    phone
    firstName
    lastName
    currentLocation {
      latitude
      longitude
      address
    }
  }
}
```

### Получить историю местоположений водителя
```graphql
query GetDriverLocationHistory($driverId: UUID!, $limit: Int, $offset: Int) {
  driver(id: $driverId) {
    id
    firstName
    lastName
    locationHistory(limit: $limit, offset: $offset) {
      id
      latitude
      longitude
      speed
      bearing
      recordedAt
      isHighAccuracy
    }
  }
}
```

### Получить статистику по водителю
```graphql
query GetDriverStats($driverId: UUID!) {
  driver(id: $driverId) {
    id
    firstName
    lastName
    ratingStats {
      averageRating
      totalRatings
      ratingDistribution {
        one
        two
        three
        four
        five
      }
      criteriaAverages
      percentile95
      percentile90
    }
  }
  
  locationStats(driverId: $driverId) {
    totalPoints
    distanceTraveled
    averageSpeed
    maxSpeed
    timeSpan
  }
  
  shiftStats(driverId: $driverId) {
    totalShifts
    activeShifts
    completedShifts
    totalHours
    totalEarnings
    avgHourlyRate
  }
}
```

## Основные мутации (Mutations)

### Создать водителя
```graphql
mutation CreateDriver($input: CreateDriverInput!) {
  createDriver(input: $input) {
    id
    phone
    email
    firstName
    lastName
    status
    createdAt
  }
}
```

### Обновить данные водителя
```graphql
mutation UpdateDriver($id: UUID!, $input: UpdateDriverInput!) {
  updateDriver(id: $id, input: $input) {
    id
    phone
    email
    firstName
    lastName
    updatedAt
  }
}
```

### Изменить статус водителя
```graphql
mutation ChangeDriverStatus($id: UUID!, $status: Status!) {
  changeDriverStatus(id: $id, status: $status) {
    id
    status
    updatedAt
  }
}
```

### Обновить местоположение водителя
```graphql
mutation UpdateDriverLocation($driverId: UUID!, $input: LocationUpdateInput!) {
  updateDriverLocation(driverId: $driverId, input: $input) {
    id
    driverId
    latitude
    longitude
    speed
    bearing
    recordedAt
    isValidLocation
  }
}
```

### Добавить рейтинг водителю
```graphql
mutation AddDriverRating($driverId: UUID!, $orderId: UUID, $customerId: UUID, $input: RatingInput!) {
  addDriverRating(driverId: $driverId, orderId: $orderId, customerId: $customerId, input: $input) {
    id
    driverId
    orderId
    rating
    comment
    criteriaScores
    isVerified
    createdAt
  }
}
```

### Начать смену
```graphql
mutation StartShift($driverId: UUID!, $input: ShiftStartInput) {
  startShift(driverId: $driverId, input: $input) {
    id
    driverId
    vehicleId
    status
    startTime
    startLocation {
      latitude
      longitude
    }
  }
}
```

### Завершить смену
```graphql
mutation EndShift($driverId: UUID!, $input: ShiftEndInput) {
  endShift(driverId: $driverId, input: $input) {
    id
    driverId
    status
    startTime
    endTime
    duration
    totalTrips
    totalDistance
    totalEarnings
    earningsPerHour
  }
}
```

## Примеры использования

### JavaScript (с Apollo Client)
```javascript
import { gql, useQuery, useMutation } from '@apollo/client';

// Получить водителя
const GET_DRIVER = gql`
  query GetDriver($id: UUID!) {
    driver(id: $id) {
      id
      firstName
      lastName
      phone
      status
      currentRating
      currentLocation {
        latitude
        longitude
      }
    }
  }
`;

function DriverProfile({ driverId }) {
  const { loading, error, data } = useQuery(GET_DRIVER, {
    variables: { id: driverId }
  });

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error.message}</div>;

  return (
    <div>
      <h2>{data.driver.firstName} {data.driver.lastName}</h2>
      <p>Phone: {data.driver.phone}</p>
      <p>Status: {data.driver.status}</p>
      <p>Rating: {data.driver.currentRating}</p>
    </div>
  );
}

// Создать водителя
const CREATE_DRIVER = gql`
  mutation CreateDriver($input: CreateDriverInput!) {
    createDriver(input: $input) {
      id
      firstName
      lastName
      phone
      status
    }
  }
`;

function CreateDriverForm() {
  const [createDriver, { data, loading, error }] = useMutation(CREATE_DRIVER);

  const handleSubmit = (formData) => {
    createDriver({
      variables: {
        input: {
          phone: formData.phone,
          email: formData.email,
          firstName: formData.firstName,
          lastName: formData.lastName,
          birthDate: formData.birthDate,
          licenseNumber: formData.licenseNumber,
          licenseExpiry: formData.licenseExpiry,
          passportSeries: formData.passportSeries,
          passportNumber: formData.passportNumber
        }
      }
    });
  };

  // ... остальная логика компонента
}
```

### Python (с requests)
```python
import requests
import json

# GraphQL endpoint
GRAPHQL_URL = "http://localhost:8001/graphql"

def get_driver(driver_id):
    query = """
    query GetDriver($id: UUID!) {
        driver(id: $id) {
            id
            firstName
            lastName
            phone
            status
            currentRating
        }
    }
    """
    
    response = requests.post(GRAPHQL_URL, json={
        'query': query,
        'variables': {'id': driver_id}
    })
    
    return response.json()

def create_driver(driver_data):
    mutation = """
    mutation CreateDriver($input: CreateDriverInput!) {
        createDriver(input: $input) {
            id
            firstName
            lastName
            phone
            status
        }
    }
    """
    
    response = requests.post(GRAPHQL_URL, json={
        'query': mutation,
        'variables': {'input': driver_data}
    })
    
    return response.json()

# Использование
driver = get_driver("550e8400-e29b-41d4-a716-446655440000")
print(f"Driver: {driver['data']['driver']['firstName']} {driver['data']['driver']['lastName']}")

new_driver = create_driver({
    'phone': '+1234567890',
    'email': 'test@example.com',
    'firstName': 'John',
    'lastName': 'Doe',
    'birthDate': '1990-01-01T00:00:00Z',
    'licenseNumber': 'DL123456',
    'licenseExpiry': '2030-01-01T00:00:00Z',
    'passportSeries': '1234',
    'passportNumber': '567890'
})
```

### cURL примеры
```bash
# Получить водителя
curl -X POST http://localhost:8001/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query GetDriver($id: UUID!) { driver(id: $id) { id firstName lastName phone status } }",
    "variables": { "id": "550e8400-e29b-41d4-a716-446655440000" }
  }'

# Создать водителя
curl -X POST http://localhost:8001/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation CreateDriver($input: CreateDriverInput!) { createDriver(input: $input) { id firstName lastName phone status } }",
    "variables": {
      "input": {
        "phone": "+1234567890",
        "email": "test@example.com", 
        "firstName": "John",
        "lastName": "Doe",
        "birthDate": "1990-01-01T00:00:00Z",
        "licenseNumber": "DL123456",
        "licenseExpiry": "2030-01-01T00:00:00Z",
        "passportSeries": "1234",
        "passportNumber": "567890"
      }
    }
  }'

# Обновить местоположение
curl -X POST http://localhost:8001/graphql \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation UpdateLocation($driverId: UUID!, $input: LocationUpdateInput!) { updateDriverLocation(driverId: $driverId, input: $input) { id latitude longitude } }",
    "variables": {
      "driverId": "550e8400-e29b-41d4-a716-446655440000",
      "input": {
        "latitude": 55.7558,
        "longitude": 37.6176,
        "speed": 60.0
      }
    }
  }'
```

## Обработка ошибок

GraphQL API возвращает ошибки в стандартном формате GraphQL:

```json
{
  "data": null,
  "errors": [
    {
      "message": "driver not found",
      "locations": [
        {
          "line": 2,
          "column": 3
        }
      ],
      "path": ["driver"],
      "extensions": {
        "code": "DRIVER_NOT_FOUND"
      }
    }
  ]
}
```

Основные коды ошибок:
- `DRIVER_NOT_FOUND` - водитель не найден
- `INVALID_INPUT` - некорректные входные данные
- `DRIVER_NOT_AVAILABLE` - водитель недоступен
- `PERMISSION_DENIED` - доступ запрещен
- `INTERNAL_ERROR` - внутренняя ошибка сервера

## Запуск и тестирование

1. Запустите сервер:
```bash
cd driver-service
go run cmd/server/main.go
```

2. Откройте GraphQL Playground в браузере:
```
http://localhost:8001/playground
```

3. Запустите тесты:
```bash
# Unit тесты
go test ./internal/interfaces/graphql/resolver/...

# Интеграционные тесты
go test ./tests/integration/...
```

## Дополнительные возможности

### Подписки (Subscriptions)
GraphQL API поддерживает real-time подписки для получения обновлений в реальном времени:

```graphql
subscription DriverLocationUpdated($driverId: UUID!) {
  driverLocationUpdated(driverId: $driverId) {
    id
    latitude
    longitude
    speed
    recordedAt
  }
}
```

### Пагинация
Все списочные запросы поддерживают пагинацию через параметры `limit` и `offset`.

### Фильтрация
Большинство запросов поддерживают расширенные фильтры для точного поиска данных.

### Кэширование
API поддерживает автоматическое кэширование запросов и persisted queries для повышения производительности.