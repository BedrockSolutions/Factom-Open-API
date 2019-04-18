# Factom Open API

## Installation (developer release)

Run postgres DB container
```bash
docker run -d --name foa-db postgres
```

Create a folder (e.g. ~/.foa) for config and download config template into it
```bash
mkdir ~/.foa
curl -o ~/.foa/config.yaml https://raw.githubusercontent.com/DeFacto-Team/Factom-Open-API/master/config.yaml.EXAMPLE
```

Edit config & fill Es address (you can also change other params)
```bash
nano ~/.foa/config.yaml
```

Run Open API container
```bash
docker run -d -p 8081:8081 --name factom-open-api --link foa-db -v ~/.foa:/home/app/values defactoteam/factom-open-api:1.0.0-rc1
```

**Congratulations!**
Your Factom Open API available at http://localhost:8081

## User management

For access & work with Factom Open API you need to create user(s).
In the next version the user management will be possible via admin endpoint and Web UI, but for current release we developed the admin binary.

The binary is embedded into Open API container, so you can run it via terminal:
```bash
docker exec -ti factom-open-api ./user create anton
```
You will see access key into terminal.
By default, new users **are enabled** and **have no writes limit**.

You can manage users with additional binary commands:
```bash
# create user `anton` and generate API access key
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml create anton

# disable access to API for user `anton`
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml disable anton

# enable access to API for user `anton`
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml enable anton

# delete user `anton`
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml delete anton

# rotate API access key for user `anton`
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml rotate-key anton

# set writes limit for user `anton` to `1000` // 0 for unlimited
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml set-limit anton 1000

# show users, API keys & params
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml ls

# show help
docker exec -ti factom-open-api ./user -c=/home/app/values/config.yaml help
```
