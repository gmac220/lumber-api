# lumber-api
A minimal Go API for managing lumber inventory and orders.

## Tech Stack

- Go (standard library)
- PostgreSQL
- database/sql (no ORM)
- Docker Compose

---

## Setup Instructions

### 1. Clone repository

### 2. When running for the first time

```bash
go mod tiny
```

### 3. Run with Docker
```bash
docker compose up --build
```

### 4. Run api
```bash
docker compose up api
```

### 5. To check db 
```bash
docker exec -it lumber-api-db-1 psql -U lumber -d lumber
```

### 6. Add inventory and place orders
```bash
curl -X POST http://localhost:8080/products \
-H "Content-Type: application/json" \
-d '{
  "sku": "2X4-8FT",
  "quantity_on_hand": 500
}'


curl -X POST http://localhost:8080/orders \
-H "Content-Type: application/json" \
-d '{
  "product_id": 1,
  "quantity": 100
}'
```
---
Overselling is prevented via:

- Atomic inventory update:
```bash
UPDATE products
SET quantity_on_hand = quantity_on_hand - $1
WHERE id = $2 AND quantity_on_hand >= $1
```

- Serializable transaction isolation
- Checking RowsAffected()

Assumptions
- Orders do not need cancellation
- No pagination required
- No authentication required
- Product inventory managed only via orders

Design Decisions
Atomic update
- We use atomic UPDATE with condition to reduce lock contention
- Simpler and safer pattern
- Avoids race conditions

Why Serializable?
- Provides strongest safety

No ORM:
- Direct SQL improves clarity
- Makes transaction logic explicit
- Better demonstrates understanding of isolation & constraints

Improvements with more time:
- Product lookup by SKU endpoint
- Adding inventory to an existing one