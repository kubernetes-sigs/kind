##At first machine
# 1. 生成 key（如果没有）
[ -f ~/.ssh/id_ed25519 ] || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519

# 2. 看你的 public key
cat ~/.ssh/id_ed25519.pub


## At second machine
mkdir -p ~/.ssh
chmod 700 ~/.ssh

# 把刚才那一行粘贴进去（替换 <KEY>）
echo '<KEY>' >> ~/.ssh/authorized_keys

chmod 600 ~/.ssh/authorized_keys


## At 1st machine
ssh ecotype-48 hostname

docker context create ecotype-48 \
    --docker host=ssh://root@ecotype-48.nantes.grid5000.fr

docker --context=ecotype-48 ps