import requests
from utils import API_GATEWAY_URL, WithDeletePassport


def test_e2e_posts():
    return
    login = "testuser"
    email = "mail@example.com"
    with WithDeletePassport(
        f"DELETE FROM users WHERE login = '{login}' OR email = '{email}'"
    ):
        print("Creating user...")
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
        print("User created successfully.")
        print("Logging in...")
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
        print("Logged in successfully.")
        response = requests.post(
            f"{API_GATEWAY_URL}/posts",
            headers={"Authorization": token},
            json={
                "title": "Test Post",
                "description": "This is a test post.",
                "is_private": False,
            },
        )
        assert response.status_code == 201, response.text
        print(response.text)
