## Connection Pools, and why sizing matter here
>>You don't open a database connection per request. TCP handshake + TLS + Postgres auth is expensive (milliseconds), and Postgres allocates a whole backend process per connection — it's not cheap on their side either. So you keep a pool of open connections and hand them out. Standard stuff. But VaultPay has a specific hazard, and it's worth seeing before you write the ledger. The pool-exhaustion deadlock. Recall §8.4: a transfer does SELECT ... FOR UPDATE on the sender's balance row. If a second transfer hits the same wallet, it blocks — waiting for the first to commit. Now notice: that blocked transaction is still holding a pool connection while it waits. Scale that up. Pool size 5. Ten concurrent transfers on a hot wallet. Five grab connections and block on the lock. The other five can't even get a connection to try. If the lock-holder needs a connection for anything else, it can't get one — everything is waiting on everything. Service wedges. 

### The mitigations, and we'll use all three:

```Pool big enough that lock waiters don't starve the pool.
lock_timeout — a transaction that can't get its row lock within N ms gives up rather than waiting forever.
Short transactions. Never do slow work (an HTTP call, a Claude API call) inside a DB transaction. GoHunt's scorer taught you this instinct; here it's load-bearing.```