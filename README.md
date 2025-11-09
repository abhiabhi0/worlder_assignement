# Sensor Data Generator & Ingestion System (Microservice A & B)

This project implements a **distributed sensor data system** using **Go**, **Echo**, **gRPC**, and **MySQL**, following a microservice architecture.

The assignment requires two services:

* **Microservice A** → Generates sensor readings
* **Microservice B** → Receives, stores, retrieves, edits, deletes readings

Below is a simple explanation of how both services work and how to run everything locally.

---

## Microservice A — Sensor Generator

Microservice A is responsible for **simulating real sensor devices**.

###  What Microservice A Does

* Continuously **generates sensor readings**.
* Each reading contains:

  * **value** (float)
  * **sensor_type** (temperature, humidity, etc.)
  * **id1** (capital letter, e.g., “A”)
  * **id2** (integer, e.g., 1)
  * **timestamp** (UTC, in milliseconds)
* Sends every generated reading to Microservice B over **gRPC**.
* Exposes **one REST endpoint** to change the data generation **frequency** while the service is running.

###  Why This Design

This simulates real IoT sensors where:

* Each sensor instance has fixed identity (`sensor_type`, `id1`, `id2`)
* Sensors generate data periodically
* Frequency might be adjusted dynamically (like changing sampling rate)

###  How the functionality is implemented

* A background **generator loop** produces readings at a configurable interval.
* A **sink function** pushes every reading to Microservice B using a gRPC stream.
* Echo REST endpoint:

  ```
  PUT /api/v1/frequency
  ```

  allows changing frequency by:

  * specifying readings per second (`hz`)
  * or specifying millisecond period (`period_ms`)

###  When running multiple A instances

You can simulate multiple different sensors by changing constants:

```go
SENSOR_TYPE = "humidity"
ID1 = "B"
ID2 = 2
PORT = ":8082"
```

Each instance represents **one real sensor**.

---

## Microservice B — Ingest, Store & Query Sensor Readings

Microservice B acts as the **backend server** that receives sensor data, stores it, and exposes REST APIs for clients to query or manage the data.

###  What Microservice B Does

1. **Receives sensor data** from Microservice A using **gRPC streaming**
2. **Stores** all readings into MySQL
3. Exposes **REST APIs** to:

   * Retrieve by ID combination
   * Retrieve by duration
   * Retrieve by ID + duration
   * Edit existing values
   * Delete existing values
   * Paginate results
4. All REST endpoints are protected by **JWT authentication**

###  Why This Design

This service acts like a **central data collector** for multiple sensors.
It allows external applications to:

* Fetch historical data
* Filter based on time
* Clean incorrect sensor entries
* Update values
* Delete old or unwanted data

This matches real-world ingestion pipelines in IoT and telemetry platforms.

###  How the functionality is implemented

* **gRPC Ingest Service** (`StreamReadings`)
  A opens a persistent stream and sends readings continuously.
* **MySQL storage**
  Each reading is inserted with:

  * value
  * sensor_type
  * id1
  * id2
  * timestamp
* **REST API on Echo**

  * Queries use flexible filters.
  * Pagination is implemented using `page` and `limit` parameters.
  * Edit and delete are done using POST bodies with filters.
* **JWT auth**

  * `/auth/login` gives viewer/admin tokens
  * Viewers can only **read**
  * Admins can **edit** and **delete**

---

##  How to Set Up and Run Everything

This section contains the exact commands to run the system end-to-end.

---

###  1) Start MySQL

WSL/Ubuntu:

```bash
sudo service mysql start
```

Create DB:

```bash
mysql -u root -p
```

Inside MySQL:

```sql
CREATE DATABASE sensors CHARACTER SET utf8mb4;
```

---

###  2) Generate protobuf code (only for first setup)

```bash
cd services/ingest-b
protoc --go_out=. --go-grpc_out=. proto/sensor.proto
```

---

###  3) Run Microservice B (receiver + REST API)

```bash
cd services/ingest-b
go run ./cmd/server
```

* REST API → `http://localhost:8080`
* gRPC → `localhost:9090`

---

###  4) Run Microservice A (generator → sends to B)

```bash
cd services/generator-a
go mod tidy
go run ./cmd/server
```

REST endpoint for A (change frequency):

```
http://localhost:8081/api/v1/frequency
```

#### Change frequency example:

```bash
curl -X PUT http://localhost:8081/api/v1/frequency \
  -H "Content-Type: application/json" \
  -d '{"hz": 5}'
```

---

##  5) Use Microservice B’s REST APIs

### Step 1: Login (viewer)

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "viewer", "password": "viewer123"}'
```

### Step 2: Query readings

**Retrieve by ID:**

```bash
curl "http://localhost:8080/api/v1/readings?id1=A&id2=1&limit=20&page=1" \
  -H "Authorization: Bearer <TOKEN>"
```

**Retrieve by duration:**

```bash
curl "http://localhost:8080/api/v1/readings?from=2025-01-01T00:00:00Z&to=2025-01-02T00:00:00Z" \
  -H "Authorization: Bearer <TOKEN>"
```

---

### For editing/deleting: obtain admin token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

**Edit matching values:**

```bash
curl -X PATCH http://localhost:8080/api/v1/readings \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"filter":{"id1":"A","id2":1},"set":{"value":99.9}}'
```

**Delete matching values:**

```bash
curl -X DELETE http://localhost:8080/api/v1/readings \
  -H "Authorization: Bearer <ADMIN_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"id1":"A","id2":1}'
```

---
