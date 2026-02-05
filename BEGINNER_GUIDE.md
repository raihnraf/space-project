# OrbitStream: Beginner's Guide

> A comprehensive guide for understanding the project from scratch and testing it manually.

---

## Table of Contents

1. [What is This Project?](#what-is-this-project)
2. [Key Concepts Explained](#key-concepts-explained)
3. [Architecture Deep Dive](#architecture-deep-dive)
4. [Data Flow Walkthrough](#data-flow-walkthrough)
5. [Testing with Postman](#testing-with-postman)
6. [Understanding the Code](#understanding-the-code)

---

## What is This Project?

### Simple Analogy

Imagine you're **NASA** and you have **100 satellites** orbiting Earth. Each satellite needs to send you information about:
- 🔋 **Battery level** (are they running out of power?)
- 💾 **Storage usage** (is the hard drive full?)
- 📡 **Signal strength** (can they still talk to Earth?)

Now imagine all 100 satellites sending this data **10 times per second**. That's **1,000 messages per second**! You need a system that can:
1. **Receive** all this data quickly
2. **Store** it efficiently
3. **Detect problems** (like low battery) automatically
4. **Show pretty charts** so engineers can monitor everything

**OrbitStream** is that system - a practice project that simulates this scenario.

### Real-World Applications

This same pattern is used for:
- 🚗 **Tesla cars** sending diagnostic data
- 🏠 **Smart home devices** (thermostats, cameras)
- 🏭 **Factory sensors** monitoring machines
- 📱 **Mobile apps** sending usage analytics
- 🌡️ **Weather stations** reporting conditions

---

## Key Concepts Explained

### 1. Telemetry

**Definition:** Data sent from a remote device to a central system.

```
Satellite ----> "My battery is at 85%" ----> Ground Station
```

In our project, each telemetry point contains:
```json
{
  "satellite_id": "SAT-0001",
  "timestamp": "2024-01-15T10:30:00Z",
  "battery_charge_percent": 85.5,
  "storage_usage_mb": 45000.0,
  "signal_strength_dbm": -55.0
}
```

### 2. High-Throughput

**Definition:** Handling LOTS of data very fast.

| Comparison | Speed |
|------------|-------|
| Loading a webpage | ~1 request |
| Sending an email | ~1 request |
| **OrbitStream** | **10,000 requests per second** |

This is like the difference between:
- 🐢 One person walking through a door vs
- 🚀 10,000 people running through 100 doors simultaneously

### 3. Anomaly Detection

**Definition:** Automatically finding unusual/bad data.

```
Normal:     Battery = 85%  ✅
Anomaly:    Battery = 5%   ⚠️  [ALERT: Low battery!]

Normal:     Signal = -55 dBm  ✅
Anomaly:    Signal = -115 dBm ⚠️  [ALERT: Weak signal!]
```

### 4. Time-Series Database

**Definition:** A database optimized for data that changes over time.

Regular Database (like MySQL):
```
User: John, Email: john@example.com  [Current state only]
```

Time-Series Database (like TimescaleDB):
```
10:00 - Battery: 90%
10:01 - Battery: 89%
10:02 - Battery: 88%
[History preserved!]
```

### 5. Batch Processing

**Definition:** Collecting many small items and processing them together.

❌ **Without Batching:**
```
Receive 1 item → Save to database
Receive 1 item → Save to database
Receive 1 item → Save to database
... (10,000 times per second!)
```

✅ **With Batching (what we do):**
```
Receive 1,000 items → Collect in buffer → Save all at once
```

This is like:
- ❌ Going to the store for 1 item, 100 times
- ✅ Making a shopping list and going once

---

## Architecture Deep Dive

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      ORBITSTREAM SYSTEM                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────┐      HTTP POST       ┌─────────────┐ │
│  │ Python Simulator │ ───────────────────> │ Go Service  │ │
│  │ (Fake Satellites)│   "Here's my data!"   │ (Receiver)  │ │
│  └──────────────────┘                      └──────┬──────┘ │
│                                                    │         │
│                                                    │ Batch   │
│                                                    │ Insert  │
│                                                    ▼         │
│                                             ┌─────────────┐ │
│                                             │ TimescaleDB │ │
│                                             │ (Database)  │ │
│                                             └──────┬──────┘ │
│                                                    │         │
│                                                    │ Query   │
│                                                    ▼         │
│                                             ┌─────────────┐ │
│                                             │   Grafana   │ │
│                                             │ (Dashboard) │ │
│                                             └─────────────┘ │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Each Component Explained

#### 1. Python Simulator (`python-simulator/`)

**What it does:** Pretends to be 100 satellites sending data.

**Think of it as:** A robot that presses "send" on 100 phones simultaneously, over and over.

**Key Files:**
- `satellite_sim.py` - Main script that creates "fake satellites"
- `generators/telemetry_gen.py` - Creates realistic fake data
- `config.py` - Settings (how many satellites, how fast, etc.)

**Code Example:**
```python
# This creates a "satellite" that sends data every 10ms
satellite = Satellite(id="SAT-0001", generator=TelemetryGenerator())
```

#### 2. Go Ingestion Service (`go-service/`)

**What it does:** Receives HTTP requests and saves data to database.

**Think of it as:** A mailroom that receives letters, sorts them, and puts them in filing cabinets.

**Key Features:**
- **HTTP Server:** Listens on port 8080 for incoming data
- **Batching:** Collects 1000 data points before saving
- **Anomaly Detection:** Checks if data looks "suspicious"

**Key Files:**
- `main.go` - Starts the HTTP server
- `handlers/telemetry.go` - Handles HTTP requests
- `db/batch.go` - Batches and saves data
- `db/connection.go` - Connects to TimescaleDB

**Endpoints:**
```
GET  /health              → "Yes, I'm alive!"
POST /telemetry           → "Thanks for the data point!"
POST /telemetry/batch     → "Thanks for the batch of data!"
```

#### 3. TimescaleDB (PostgreSQL extension)

**What it does:** Stores time-series data efficiently.

**Think of it as:** A super-organized filing system designed for "when did X happen?" questions.

**Key Features:**
- **Hypertables:** Automatically partitions data by time
- **Compression:** Makes old data smaller
- **Retention:** Can auto-delete data older than X days

**Table Structure:**
```sql
CREATE TABLE telemetry (
    time TIMESTAMPTZ,           -- When did it happen?
    satellite_id TEXT,          -- Which satellite?
    battery_charge_percent FLOAT,
    storage_usage_mb FLOAT,
    signal_strength_dbm FLOAT,
    is_anomaly BOOLEAN          -- Was this flagged as weird?
);
```

#### 4. Grafana

**What it does:** Shows pretty charts and graphs.

**Think of it as:** A TV screen in the control room showing satellite status.

**What you can see:**
- Real-time throughput (points per second)
- Battery levels over time
- Which satellites have problems
- Total data stored

---

## Data Flow Walkthrough

Let's trace what happens when **one piece of data** travels through the system:

### Step 1: Python Simulator Creates Data

```python
# Inside TelemetryGenerator
data = {
    "satellite_id": "SAT-0042",
    "battery_charge_percent": 85.5,
    "storage_usage_mb": 45000.0,
    "signal_strength_dbm": -55.0
}
```

### Step 2: Python Sends HTTP Request

```python
# Inside Satellite.send_telemetry()
import aiohttp

async with session.post(
    "http://localhost:8080/telemetry",
    json=data
) as response:
    # Wait for response
```

**What this looks like:**
```
POST http://localhost:8080/telemetry
Content-Type: application/json

{
  "satellite_id": "SAT-0042",
  "timestamp": "2024-01-15T10:30:00.123Z",
  "battery_charge_percent": 85.5,
  "storage_usage_mb": 45000.0,
  "signal_strength_dbm": -55.0
}
```

### Step 3: Go Service Receives Request

```go
// Inside handlers/telemetry.go
func (h *TelemetryHandler) HandleTelemetry(c *gin.Context) {
    var point models.TelemetryPoint
    
    // Parse JSON from request body
    c.ShouldBindJSON(&point)
    
    // Add to batch for later saving
    h.batchProcessor.Add(point)
    
    // Respond immediately (don't wait for database)
    c.JSON(202, {"status": "accepted"})
}
```

### Step 4: Anomaly Detection Happens

```go
// Inside db/batch.go
func (bp *BatchProcessor) detectAnomaly(point TelemetryPoint) bool {
    // Is battery too low?
    if point.BatteryChargePercent < 10.0 {
        log.Println("ANOMALY: Battery critically low!")
        return true
    }
    
    // Is storage too high?
    if point.StorageUsageMB > 95000.0 {
        log.Println("ANOMALY: Storage critically high!")
        return true
    }
    
    // Is signal too weak?
    if point.SignalStrengthDBM < -100.0 {
        log.Println("ANOMALY: Signal critically weak!")
        return true
    }
    
    return false
}
```

### Step 5: Data is Batched

```go
// Data waits in a buffer (in memory)
buffer := []TelemetryPoint{
    {SatelliteID: "SAT-0001", ...},
    {SatelliteID: "SAT-0002", ...},
    // ... up to 1000 items
}
```

### Step 6: Batch is Saved to Database

```go
// When buffer reaches 1000 items OR 1 second passes:
func (bp *BatchProcessor) insertBatch(batch []TelemetryPoint) {
    // Insert all 1000 items in one transaction
    INSERT INTO telemetry VALUES 
        (...), (...), (...)  -- 1000 times
}
```

### Step 7: Grafana Queries the Data

```sql
-- Grafana runs this query every few seconds
SELECT 
    time_bucket('1 minute', time) as minute,
    COUNT(*) as points_per_minute
FROM telemetry
WHERE time > NOW() - INTERVAL '1 hour'
GROUP BY minute
ORDER BY minute;
```

### Step 8: You See a Chart! 📊

---

## Testing with Postman

[The rest of the Postman guide continues...]
---

## Testing with Postman

### What is Postman?

Postman is a tool for testing APIs (Application Programming Interfaces). Think of it like a "browser for APIs" - instead of viewing webpages, you send requests and see raw responses.

**Download:** https://www.postman.com/downloads/

---

### Prerequisites

Before testing with Postman, start the services:

```bash
# 1. Start the database and Grafana
docker compose up -d timescaledb grafana

# 2. Wait 30 seconds for database to be ready
# Check logs: docker compose logs -f timescaledb

# 3. Start the Go service
docker compose up -d go-service

# 4. Verify it's running
curl http://localhost:8080/health
```

---

### Test 1: Health Check (GET Request)

**Purpose:** Verify the service is running.

**Steps:**

1. Open Postman
2. Click **"New"** → **"HTTP Request"**
3. Set method to **GET**
4. Enter URL: `http://localhost:8080/health`
5. Click **"Send"**

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00.123456789Z"
}
```

**What this means:** ✅ The service is alive and ready to accept data!

**Screenshot of settings:**
```
┌────────────────────────────────────────┐
│  GET  http://localhost:8080/health    │
│  [Send]                                │
│                                        │
│  Headers (0)                           │
│  Body    (none)                        │
└────────────────────────────────────────┘
```

---

### Test 2: Send Single Telemetry Point (POST Request)

**Purpose:** Send one satellite data point.

**Steps:**

1. Create a new request
2. Set method to **POST**
3. Enter URL: `http://localhost:8080/telemetry`
4. Click **"Body"** tab
5. Select **"raw"** and choose **"JSON"** from dropdown
6. Paste this JSON:

```json
{
  "satellite_id": "SAT-TEST-01",
  "battery_charge_percent": 85.5,
  "storage_usage_mb": 45000.0,
  "signal_strength_dbm": -55.0
}
```

7. Click **"Send"**

**Expected Response:**
```json
{
  "status": "accepted",
  "satellite_id": "SAT-TEST-01"
}
```

**Status Code:** `202 Accepted`

**What this means:** ✅ The data was received and will be saved to the database.

**Screenshot of settings:**
```
┌────────────────────────────────────────┐
│  POST  http://localhost:8080/telemetry │
│  [Send]                                │
│                                        │
│  Body    [raw] [JSON]                  │
│  ┌─────────────────────────────────┐   │
│  │ {                               │   │
│  │   "satellite_id": "SAT-TEST-01│   │
│  │   "battery_charge_percent": 85.5│  │
│  │   "storage_usage_mb": 45000.0,  │   │
│  │   "signal_strength_dbm": -55.0  │   │
│  │ }                               │   │
│  └─────────────────────────────────┘   │
└────────────────────────────────────────┘
```

---

### Test 3: Send Batch of Telemetry Points

**Purpose:** Send multiple data points at once (more efficient).

**Steps:**

1. Create a new request
2. Set method to **POST**
3. Enter URL: `http://localhost:8080/telemetry/batch`
4. Click **"Body"** tab
5. Select **"raw"** and choose **"JSON"**
6. Paste this JSON (array of objects):

```json
[
  {
    "satellite_id": "SAT-BATCH-01",
    "battery_charge_percent": 90.0,
    "storage_usage_mb": 30000.0,
    "signal_strength_dbm": -50.0
  },
  {
    "satellite_id": "SAT-BATCH-02",
    "battery_charge_percent": 75.5,
    "storage_usage_mb": 55000.0,
    "signal_strength_dbm": -60.0
  },
  {
    "satellite_id": "SAT-BATCH-03",
    "battery_charge_percent": 60.0,
    "storage_usage_mb": 80000.0,
    "signal_strength_dbm": -70.0
  }
]
```

7. Click **"Send"**

**Expected Response:**
```json
{
  "status": "accepted",
  "count": 3
}
```

**Status Code:** `202 Accepted`

**What this means:** ✅ All 3 data points were received.

---

### Test 4: Trigger an Anomaly Alert

**Purpose:** See what happens when data looks suspicious.

**Steps:**

1. Create a new POST request to `http://localhost:8080/telemetry`
2. Send this JSON (very low battery):

```json
{
  "satellite_id": "SAT-DYING-01",
  "battery_charge_percent": 5.0,
  "storage_usage_mb": 45000.0,
  "signal_strength_dbm": -55.0
}
```

**What to check:**
1. ✅ Postman shows `202 Accepted` response
2. ✅ Check Go service logs:
   ```bash
   docker compose logs go-service
   ```
   You should see:
   ```
   ANOMALY: Satellite SAT-DYING-01 battery critically low: 5.00%
   ```

**Try these other anomalies:**

**Storage Full:**
```json
{
  "satellite_id": "SAT-FULL-01",
  "battery_charge_percent": 85.0,
  "storage_usage_mb": 98000.0,
  "signal_strength_dbm": -55.0
}
```

**Weak Signal:**
```json
{
  "satellite_id": "SAT-WEAK-01",
  "battery_charge_percent": 85.0,
  "storage_usage_mb": 45000.0,
  "signal_strength_dbm": -115.0
}
```

---

### Test 5: Send Invalid Data (Error Handling)

**Purpose:** See how the API handles bad requests.

**Test A: Invalid JSON**

1. POST to `http://localhost:8080/telemetry`
2. Send plain text (not JSON):
```
this is not json
```

**Expected Response:**
```json
{
  "error": "invalid character 'h' in literal true (expecting 'r')"
}
```
**Status Code:** `400 Bad Request`

**Test B: Missing Required Fields**

1. POST to `http://localhost:8080/telemetry`
2. Send empty object:
```json
{}
```

**Expected Response:** `202 Accepted` (Go uses zero values for missing fields)

---

### Test 6: Create a Postman Collection

Instead of creating requests one by one, create a reusable collection:

1. Click **"Collections"** → **"Create Collection"**
2. Name it: `OrbitStream API`
3. Click **"Add Request"** for each test:

| Request Name | Method | URL | Body |
|--------------|--------|-----|------|
| Health Check | GET | `http://localhost:8080/health` | None |
| Send Telemetry | POST | `http://localhost:8080/telemetry` | JSON (single point) |
| Send Batch | POST | `http://localhost:8080/telemetry/batch` | JSON (array) |
| Anomaly - Low Battery | POST | `http://localhost:8080/telemetry` | JSON (battery: 5) |

Now you can run all tests with one click!

---

### Test 7: View Data in Grafana

After sending data with Postman:

1. Open http://localhost:3000
2. Login: `admin` / `admin`
3. Click **"Explore"** (compass icon)
4. Select **"timescaledb"** datasource
5. Run this query:

```sql
SELECT * FROM telemetry 
WHERE satellite_id = 'SAT-TEST-01'
ORDER BY time DESC
LIMIT 10;
```

You should see your test data!

---

## Understanding the Code

### For Python Beginners

**What's a dataclass?**
```python
from dataclasses import dataclass

@dataclass
class Satellite:
    id: str
    generator: TelemetryGenerator

# This automatically creates __init__, __repr__, etc.
sat = Satellite(id="SAT-01", generator=gen)
```

**What's async/await?**
```python
import asyncio

async def send_data():
    # "await" means "wait for this to finish without blocking"
    result = await session.post(url, json=data)
    return result

# Run async function
asyncio.run(send_data())
```

**What's a fixture in pytest?**
```python
@pytest.fixture
def telemetry_generator():
    # This creates a reusable test object
    return TelemetryGenerator()

def test_something(telemetry_generator):
    # pytest automatically provides the fixture
    result = telemetry_generator.generate_telemetry()
```

### For Go Beginners

**What's a struct?**
```go
type TelemetryPoint struct {
    SatelliteID          string    `json:"satellite_id"`
    BatteryChargePercent float64   `json:"battery_charge_percent"`
}

// Create instance
point := TelemetryPoint{
    SatelliteID: "SAT-01",
    BatteryChargePercent: 85.5,
}
```

**What's a goroutine?**
```go
// Run function in background (like a lightweight thread)
go batchProcessor.Start()

// "go" keyword makes it run concurrently
```

**What's a channel?**
```go
// A pipe for goroutines to communicate
done := make(chan bool)

// Send data into channel
done <- true

// Receive from channel
value := <-done
```

---

## Common Beginner Questions

### Q: Why do I need PYTHONPATH for tests?

**A:** Python needs to know where to find the `generators` and `config` modules. By default, it only looks in the current directory. `PYTHONPATH` tells it to also look in the project root.

### Q: Why 202 Accepted instead of 200 OK?

**A:** HTTP status codes have meanings:
- `200 OK` - Request succeeded, here's your data
- `201 Created` - Request succeeded, I created a new resource
- `202 Accepted` - Request received, I'll process it later (this is what we use because we batch data)

### Q: What's the difference between simulator and real satellites?

**A:** The simulator just generates fake data. In a real system:
- Real satellites would send actual sensor readings
- Communication would be via radio, not HTTP
- There would be encryption and authentication

### Q: Why TimescaleDB instead of regular PostgreSQL?

**A:** TimescaleDB is built on PostgreSQL but adds:
- Automatic partitioning by time
- Better compression for time-series data
- Faster queries for "last 24 hours" type questions

### Q: How do I stop everything?

```bash
# Stop all containers
docker compose down

# Stop and delete all data (CAREFUL!)
docker compose down -v
```

---

## Next Steps

Now that you understand the basics:

1. **Modify the simulator:** Change how many satellites or how fast they send data
2. **Add new telemetry fields:** Track temperature, altitude, etc.
3. **Create new Grafana dashboards:** Visualize different metrics
4. **Add authentication:** Make the API require API keys
5. **Scale up:** Run multiple Go service instances behind a load balancer

---

## Glossary

| Term | Definition |
|------|------------|
| **API** | Application Programming Interface - how programs talk to each other |
| **Batch** | Group of items processed together |
| **Docker** | Tool for running applications in containers |
| **Endpoint** | URL where an API receives requests |
| **Goroutine** | Lightweight thread in Go |
| **HTTP** | Protocol for web communication |
| **JSON** | JavaScript Object Notation - text format for data |
| **Latency** | Time delay between request and response |
| **Microservice** | Small, independent service that does one thing |
| **Mock** | Fake object for testing |
| **PostgreSQL** | Open-source relational database |
| **Request** | Message sent to a server asking for something |
| **Response** | Message sent back from a server |
| **Telemetry** | Data from remote sensors/devices |
| **Throughput** | Amount of data processed per second |
| **Time-series** | Data points ordered in time |
| **Timestamp** | Date and time when something happened |

---

**Happy Learning! 🚀**

If you have questions, check the main README.md or look at the code comments.
