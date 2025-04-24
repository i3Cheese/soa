import requests
import os

API_GATEWAY_URL = os.getenv("API_GATEWAY_URL")

resp = requests.post(
    f"{API_GATEWAY_URL}/passport/login",
    json={
        "login": "me1",
        "password": "password",
    },
)
print(resp, resp.json())
token = resp.json()['token']
resp = requests.put(
    f"{API_GATEWAY_URL}/passport/me",
    headers={"Authorization": token},
    json={
        "email": "newemail1@example.com",
        "name": "Newname",
        "surname": "newsurname",
        "date_of_birth": "1991-01-01",
        "phone_number": "+0987654322",
    },
)
print(resp, resp.json())
resp = requests.get(
    f"{API_GATEWAY_URL}/passport/me",
    headers={"Authorization": token},
)


print(resp, resp.json())
