import requests
from utils import API_GATEWAY_URL, WithDeletePassport, WithDeletePosts
from contextlib import contextmanager


@contextmanager
def register_and_login(login, email, password):
    with WithDeletePassport(f"DELETE FROM users WHERE login = '{login}' OR email = '{email}'"):
        # Register user
        requests.post(
            f"{API_GATEWAY_URL}/passport/register",
            json={
                "login": login,
                "email": email,
                "password": password,
                "name": "Test",
                "surname": "User",
                "date_of_birth": "1990-01-01",
                "phone_number": "+1234567890",
            },
        )

        # Login user
        response = requests.post(
            f"{API_GATEWAY_URL}/passport/login",
            json={"login": login, "password": password},
        )
        token = response.json()["token"]
        try:
            yield token
        finally:
            pass  # Cleanup logic if needed


def test_create_post():
    login = "testuser"
    email = "mail@example.com"
    password = "password"

    with register_and_login(login, email, password) as token:
        with WithDeletePosts(f"DELETE FROM posts WHERE title = 'Test Post'"):
            # Create post
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Test Post",
                    "description": "This is a test post.",
                    "is_private": False,
                },
            )
            assert response.status_code == 201
            data = response.json()
            assert data["title"] == "Test Post"
            assert data["description"] == "This is a test post."


def test_delete_post():
    login = "testuser"
    email = "mail@example.com"
    password = "password"

    with register_and_login(login, email, password) as token:
        with WithDeletePosts(f"DELETE FROM posts WHERE title = 'Post to Delete'"):
            # Create post
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Post to Delete",
                    "description": "This post will be deleted.",
                    "is_private": False,
                },
            )
            post_id = response.json()["post_id"]

            # Delete post
            response = requests.delete(
                f"{API_GATEWAY_URL}/posts/{post_id}",
                headers={"Authorization": token},
            )
            assert response.status_code == 200
            assert response.json()["success"]


def test_update_post():
    login = "testuser"
    email = "mail@example.com"
    password = "password"

    with register_and_login(login, email, password) as token:
        with WithDeletePosts(
            f"DELETE FROM posts WHERE title = 'Original Title' OR title = 'Updated Title'"
        ):
            # Create post
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Original Title",
                    "description": "Original Description",
                    "is_private": False,
                },
            )
            post_id = response.json()["post_id"]

            # Update post
            response = requests.put(
                f"{API_GATEWAY_URL}/posts/{post_id}",
                headers={"Authorization": token},
                json={
                    "title": "Updated Title",
                    "description": "Updated Description",
                    "is_private": True,
                },
            )
            assert response.status_code == 200
            data = response.json()
            assert data["title"] == "Updated Title"
            assert data["is_private"]


def test_get_post_by_id():
    login = "testuser"
    email = "mail@example.com"
    password = "password"

    with register_and_login(login, email, password) as token:
        with WithDeletePosts(f"DELETE FROM posts WHERE title = 'Post to Fetch'"):
            # Create post
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Post to Fetch",
                    "description": "This post will be fetched.",
                    "is_private": False,
                },
            )
            post_id = response.json()["post_id"]

            # Get post by ID
            response = requests.get(
                f"{API_GATEWAY_URL}/posts/{post_id}",
                headers={"Authorization": token},
            )
            assert response.status_code == 200
            data = response.json()
            assert data["post_id"] == post_id
            assert data["title"] == "Post to Fetch"


def test_get_posts():
    login = "testuser"
    email = "mail@example.com"
    password = "password"

    with register_and_login(login, email, password) as token:
        with WithDeletePosts(f"DELETE FROM posts WHERE title IN ('Post 1', 'Post 2')"):
            # Create posts
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Post 1",
                    "description": "First post.",
                    "is_private": False,
                },
            )
            print("Response:", response.text)
            assert response.status_code == 201, f"Failed to create Post 1: {response.text}"
            response = requests.post(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                json={
                    "title": "Post 2",
                    "description": "Second post.",
                    "is_private": False,
                },
            )
            print("Response:", response.text)
            assert response.status_code == 201, f"Failed to create Post 2: {response.text}"


            # Get posts
            response = requests.get(
                f"{API_GATEWAY_URL}/posts",
                headers={"Authorization": token},
                params={"limit": 10},
            )
            print("Response:", response.text)
            assert response.status_code == 200, f"Unexpected status code: {response.status_code}, response: {response.text}"
            data = response.json()
            assert "posts" in data, f"Response JSON does not contain 'posts': {data}"
            assert isinstance(data["posts"], list), f"'posts' is not a list: {data}"
            assert len(data["posts"]) >= 2, f"Expected at least 2 posts, got: {data}"
