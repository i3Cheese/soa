import os
import psycopg

API_GATEWAY_URL = os.getenv("API_GATEWAY_URL")
DATABASE_URL = os.getenv("DATABASE_URL")


class WithDelete:
    def __init__(self, query):
        self.query = query

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        with psycopg.connect(DATABASE_URL) as conn:
            with conn.cursor() as cur:
                cur.execute(self.query)
                conn.commit()
