./bin/reducedkind --multihost \
    --hosts "default=$MGR_IP,ecotype-35=$WORKER_IP" \
    delete demo

docker swarm leave --force
ssh root@ecotype-35.nantes.grid5000.fr "docker swarm leave --force"