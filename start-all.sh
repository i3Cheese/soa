podman network create --ignore whymsg-network 
(cd passport && podman-compose up -d)
(cd api_gateway && podman-compose up -d)
