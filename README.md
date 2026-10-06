# Fake Governmental Identity SOAP Platform

Isolated sandbox container system built to simulate national demographic and security registries via classic enterprise **SOAP/XML** payloads.

## Purpose & Specifications
Implements high-fidelity mocking of typical high-latency enterprise legacy systems leveraging XML encapsulation layers over standard HTTP transports. Used for validating registration thresholds and corporate employee locking logic.

## Tech Stack
* **Language:** Go (1.27+)
* **Protocol:** SOAP / XML over HTTP (Port `:8000`)
* **Database:** PostgreSQL 15 (Geographic/Demographic Mock Records)

## Embedded Test Cases Matrix

The database automatically provisions specific internal data matrices upon container boot:
* `twin.1@example.com`: Authorized node, profile execution status `ACTIVE`.
* `twin.2@example.com`: Throws explicit enterprise violation error: `GOV_ACCOUNT_BLOCKED`.
* `twin.3@example.com`: Completely absent from demographic registry. Throws `USER_NOT_FOUND`.

## Local Validation Sandbox

To run this sandbox independently of the primary production matrix:

```bash
docker compose up --build
```

### Postman Verification Signature
* **Method:** `POST`
* **Target Address:** `http://localhost:8000/`
* **Headers:** `Content-Type: text/xml; charset=utf-8`
* **Payload Enclosure:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://xmlsoap.org">
  <soapenv:Body>
    <fetch_secure_employee_data>
      <email>twin.2@example.com</email>
      <master_token>gos_master_secure_token_2026</master_token>
    </fetch_secure_employee_data>
  </soapenv:Body>
</soapenv:Envelope>
```
