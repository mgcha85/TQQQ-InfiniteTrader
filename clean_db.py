import sqlite3

db_path = 'data/db.sqlite'

conn = sqlite3.connect(db_path)
c = conn.cursor()

c.execute("DELETE FROM user_settings")
c.execute("DELETE FROM trade_logs")
c.execute("DELETE FROM cycle_statuses")

conn.commit()
conn.close()
print("Database cleaned successfully.")
