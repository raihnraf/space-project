im trying to run the project based on readme.md

here's what i have trying to run so far

raihnraf@Paandinar:~/space-project$ # Start TimescaleDB and Grafana
docker compose up -d timescaledb grafana

# Wait for database to initialize (check logs)
docker compose logs -f timescaledb
# Look for: "database system is ready to accept connections"
WARN[0000] /home/raihnraf/space-project/docker-compose.yml: the attribute `version` is obsolete, it will be ignored, please remove it to avoid potential confusion
WARN[0000] No services to build
[+] up 2/2
 ✔ Container orbitstream-timescaledb Running                                                                                                   0.0s
 ✔ Container orbitstream-grafana     Running                                                                                                   0.0s
WARN[0000] /home/raihnraf/space-project/docker-compose.yml: the attribute `version` is obsolete, it will be ignored, please remove it to avoid potential confusion
orbitstream-timescaledb  | The files belonging to this database system will be owned by user "postgres".
orbitstream-timescaledb  | This user must also own the server process.
orbitstream-timescaledb  |
orbitstream-timescaledb  | The database cluster will be initialized with locale "en_US.utf8".
orbitstream-timescaledb  | The default database encoding has accordingly been set to "UTF8".


raihnraf@Paandinar:~/space-project$ # Start the Go service
docker compose up -d go-service

# Verify it's running
curl http://localhost:8080/health
# Should return: {"status":"healthy","timestamp":"..."}
WARN[0000] /home/raihnraf/space-project/docker-compose.yml: the attribute `version` is obsolete, it will be ignored, please remove it to avoid potential confusion
WARN[0000] No services to build
[+] up 2/2
 ✔ Container orbitstream-timescaledb Healthy                                                                                                   0.5s
 ✔ Container orbitstream-go-service  Running                                                                                                   0.0s
{"status":"healthy","timestamp":"2026-01-16T13:44:00.722585552Z"}raihnraf@Paandinar:~/space-project$


raihnraf@Paandinar:~/space-project/python-simulator$ docker compose --profile testing up simulator
WARN[0000] /home/raihnraf/space-project/docker-compose.yml: the attribute `version` is obsolete, it will be ignored, please remove it to avoid potential confusion
WARN[0000] No services to build
[+] up 3/3
 ✔ Container orbitstream-timescaledb Running                                                                                                   0.0s
 ✔ Container orbitstream-go-service  Running                                                                                                   0.0s
 ✔ Container orbitstream-simulator   Created                                                                                                   0.2s
Attaching to orbitstream-simulator
Container orbitstream-timescaledb Waiting
Container orbitstream-timescaledb Healthy
Container orbitstream-go-service Waiting
Container orbitstream-go-service Error dependency go-service failed to start
dependency failed to start: container orbitstream-go-service is unhealthy
raihnraf@Paandinar:~/space-project/python-simulator$