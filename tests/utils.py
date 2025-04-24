import os
import psycopg

API_GATEWAY_URL = os.getenv("API_GATEWAY_URL")
PASSPORT_DATABASE_URL = os.getenv("PASSPORT_DATABASE_URL")
POSTS_DATABASE_URL = os.getenv("POSTS_DATABASE_URL")


class WithDelete:
    DATABASE_URL: str

    def __init__(self, query):
        self.query = query

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        with psycopg.connect(self.DATABASE_URL) as conn:
            with conn.cursor() as cur:
                cur.execute(self.query)
                conn.commit()


class WithDeletePassport(WithDelete):
    DATABASE_URL = PASSPORT_DATABASE_URL


class WithDeletePosts(WithDelete):
    DATABASE_URL = POSTS_DATABASE_URL
