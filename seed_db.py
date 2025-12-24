import sqlite3
import datetime

db_path = 'data/db.sqlite'

conn = sqlite3.connect(db_path)
c = conn.cursor()

# 1. Clear existing data
c.execute("DELETE FROM user_settings")
c.execute("DELETE FROM trade_logs")
c.execute("DELETE FROM cycle_statuses")

now = datetime.datetime.now()

# 2. Insert Settings
c.execute("INSERT INTO user_settings (created_at, updated_at, principal, split_count, target_rate, symbols, is_active) VALUES (?, ?, ?, ?, ?, ?, ?)",
          (now, now, 10000.0, 40, 0.10, "TQQQ", 1))

# 3. Insert Cycle Status
c.execute("INSERT INTO cycle_statuses (created_at, updated_at, symbol, current_cycle_day, total_bought_qty, avg_price, total_invested) VALUES (?, ?, ?, ?, ?, ?, ?)",
          (now, now, "TQQQ", 5, 10, 50.0, 500.0))

# 4. Insert Trade Logs
c.execute("INSERT INTO trade_logs (created_at, updated_at, date, symbol, side, type, qty, price, amount) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
          (now, now, now, "TQQQ", "BUY", "LOC", 2, 50.0, 100.0))
c.execute("INSERT INTO trade_logs (created_at, updated_at, date, symbol, side, type, qty, price, amount) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
          (now, now, now, "TQQQ", "BUY", "LOC", 8, 50.0, 400.0))

conn.commit()
conn.close()
print("Database seeded successfully.")
