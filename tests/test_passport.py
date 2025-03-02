import os
import requests
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


def test_register():
    email = "test@example.com"
    with WithDelete(f"DELETE FROM users WHERE email = '{email}'"):
        response = requests.post(
            f"{API_GATEWAY_URL}/register",
            json={
                "email": email,
                "password": "password",
                "name": "Test",
                "surname": "User",
            },
        )
        assert response.status_code == 200, response.text
        assert response.json() == {"status": "User registered successfully"}
        response = requests.post(
            f"{API_GATEWAY_URL}/login",
            json={
                "email": email,
                "password": "password",
            },
        )
        assert response.status_code == 200, response.text
