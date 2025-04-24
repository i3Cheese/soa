import requests
import psycopg
import datetime
from utils import WithDeletePassport, API_GATEWAY_URL, PASSPORT_DATABASE_URL


def test_e2e_passport():
    login = "testuser"
    email = "mail@example.com"
    with WithDeletePassport(f"DELETE FROM users WHERE login = '{login}' OR email = '{email}'"):
        response = requests.post(
            f"{API_GATEWAY_URL}/passport/register",
            json={
                "login": login,
                "email": email,
                "password": "password",
                "name": "Test",
                "surname": "User",
                "date_of_birth": "1990-01-01",
                "phone_number": "+1234567890",
            },
        )
        assert response.status_code == 200, response.text
        assert response.json() == {"status": "User registered successfully"}
        with psycopg.connect(PASSPORT_DATABASE_URL) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT user_id, login, email, name, surname, date_of_birth, phone_number, created_at, updated_at FROM users WHERE login = %s",
                    (login,),
                )
                current_user = cur.fetchone()
                assert current_user
                assert current_user[1:] == (
                    login,
                    email,
                    "Test",
                    "User",
                    datetime.date(1990, 1, 1),
                    "+1234567890",
                    current_user[7],
                    current_user[8],
                )
                created_at, updated_at = current_user[7], current_user[8]
                assert created_at == updated_at
                now = datetime.datetime.now(datetime.timezone.utc)
                created_at = created_at.replace(tzinfo=datetime.timezone.utc)
                delta = datetime.timedelta(seconds=10)
                # assert now - delta < created_at < now + delta
        response = requests.post(
            f"{API_GATEWAY_URL}/passport/login",
            json={
                "login": login,
                "password": "password",
            },
        )
        assert response.status_code == 200, response.text
        assert "token" in response.json()
        token = response.json()["token"]
        response = requests.get(
            f"{API_GATEWAY_URL}/passport/me",
            headers={"Authorization": token},
        )
        assert response.status_code == 200, response.text
        assert response.json() == {
            "login": login,
            "email": email,
            "name": "Test",
            "surname": "User",
            "date_of_birth": "1990-01-01",
            "phone_number": "+1234567890",
        }
        response = requests.put(
            f"{API_GATEWAY_URL}/passport/me",
            headers={"Authorization": token},
            json={
                "email": "newemail@example.com",
                "name": "Newname",
                "surname": "newsurname",
                "date_of_birth": "1991-01-01",
                "phone_number": "+0987654321",
            },
        )
        assert response.status_code == 200, response.text
        assert response.json() == {"status": "User updated successfully"}
        with psycopg.connect(PASSPORT_DATABASE_URL) as conn:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT user_id, login, email, name, surname, date_of_birth, phone_number, created_at, updated_at FROM users WHERE login = %s",
                    (login,),
                )
                current_user = cur.fetchone()
                assert current_user
                assert current_user[1:] == (
                    login,
                    "newemail@example.com",
                    "Newname",
                    "newsurname",
                    datetime.date(1991, 1, 1),
                    "+0987654321",
                    current_user[7],
                    current_user[8],
                )
                created_at, updated_at = current_user[7], current_user[8]
                assert created_at < updated_at
                now = datetime.datetime.now(datetime.timezone.utc)
                updated_at = updated_at.replace(tzinfo=datetime.timezone.utc)
                delta = datetime.timedelta(seconds=10)
                # assert now - delta < updated_at < now + delta
