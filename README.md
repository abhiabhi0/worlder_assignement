# Microservice A — Sensor Generator

This microservice does **exactly** what the assignment requires:

1) Generates a data stream with the following fields:  
   - `value` (float)  
   - `sensor_type` (string)  
   - `id1` (capital alphabet)  
   - `id2` (int)  
   - `timestamp` (UTC ISO format)  
2) The data generation **frequency** is **changeable through a REST API endpoint**.  
3) You can run **multiple instances** of this service, each with a **fixed** `sensor_type`.  
(Assignment: points 1, 2, and 7.) :contentReference[oaicite:0]{index=0}

---

## How it works

- The generator emits one JSON line per reading to **stdout** (this is the "stream").  
- The **only** HTTP endpoint is a **PUT** to change frequency.

---

## Run

From the repo root:

```bash
cd services/generator-a
go run ./cmd/server
