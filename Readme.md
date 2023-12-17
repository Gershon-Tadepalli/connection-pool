# Why we need Connection pooling?

Connection pooling is a way to reduce cost of opening and closing the connections by maintaining pool of open connections that can be passed to one operation to another operation.

# Advantages

- Reusing the connections for multiple operations
- Improves the overall latency of serving requests
- proper resource utilization at  server during load spike

# Intution

Maintain the pool of connections by servername and whenever operation requests for a connection it looks at the pool and pick any alive connection.
- Borrow : fetch from the pool if idle connection available
- Release: returns back connection to the pool

# Usage

- We can see different database driver libraries provide ConnectionPooling by default
  ex: pgx,ADO.net