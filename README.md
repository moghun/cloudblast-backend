# Cloud Blast Backend

## System Overview

Cloud Blast backend system is designed using the microservices architecture.

The system consists of four main components:

1. Main Service: It is responsible for hosting the HTTP server and starting the User, Tournament, and Leaderboard services. With Mux, it routes requests to handler functions that are responsible for forwarding the request to appropriate service and service responses to requesters.

2. User Service: This service handles all user-related activities like user registration, login and level progress.

3. Tournament Service: This service is responsible for managing tournaments, including their creation, updates, and player participation.

4. Leaderboard Service: This service is responsible for maintaining the tournament rankings. It uses Redis as a data cache to store and quickly retrieve the tournament rankings.

- The communication between these services and the Main Service is empowered by RabbitMQ, a highly efficient message broker. RabbitMQ queues are used to facilitate communication between services, ensuring decoupling of services and improving the system's scalability and maintainability.

- As the persistent storage, Amazon DynamoDB is used to store "User", "Tournament" and "UserInTournament" (user records in different tournaments) tables.

- CronJob is used to schedule tournaments daily.

## Setup and Execution

### Local execution

1. Start RabbitMQ and Redis services. Configure AWS Cli credentials for cloud DynamoDB access.
2.

```sh
go build -o cloudblast-backend cmd/main.go
```

3.

```sh
./cloudblast-backend
```

### Dockerized execution

1. Set "AWS_ACCESS_KEY_ID" and "AWS_SECRET_ACCESS_KEY" variables on docker-compose.yaml file

2.

```sh
docker-compose up --build
```

## API Endpoints

1. `POST /api/tournament/StartTournament`: Start a tournament - takes no parameter.

2. `POST /api/tournament/EndTournament`: End the current tournament, decides winners of the tournament - takes no parameter.

3. `POST /api/user/CreateUser`: Creates a new user - takes "username", "password" and "country" as parameters.

4. `GET /api/user/Login`: Returns a JWT token checking the password - takes "username" and "password" as parameters.

5. `GET /api/user/SearchUser`: Search for a user in the system - takes "username" as parameter.

6. `POST /api/user/UpdateProgress`: Update the progress (+100 coins and +1 progress level) of a user in a tournament - takes "username" as parameter.

7. `POST /api/tournament/EnterTournament`: Entering the current tournament as a participant - takes "username" as parameter.

8. `POST /api/tournament/UpdateScore`: Increment the score of a user in a tournament, also increment the progress of the user - takes "username" as parameter.

9. `POST /api/tournament/ClaimReward`: Claim rewards after the end of a tournament - takes "username" as parameter.

10. `GET /api/tournament/GetTournamentRank`: Gt the rank of a user in a specific tournament - takes "username" as parameter.

11. `GET /api/tournament/GetTournamentLeaderboard`: Get the leaderboard of a specific tournament, which includes the ranks and scores of all the participating users - takes "username" as parameter.

12. `GET /api/user/GetCountryLeaderboard`: Get the leaderboard of users from a specific country - takes "username" as parameter.

13. `GET /api/user/GetGlobalLeaderboard`: Get the global leaderboard of all the users in the database - takes "username" as parameter.

## Dependencies

- github.com/aws/aws-sdk-go: "v1.44.330"
- github.com/gorilla/mux: "v1.8.0"
- github.com/cespare/xxhash/v2: "v2.1.2" // indirect
- github.com/davecgh/go-spew: "v1.1.1" // indirect
- github.com/dgryski/go-rendezvous: "v0.0.0-20200823014737-9f7001d12a5f" // indirect
- github.com/dgrijalva/jwt-go: "v3.2.0+incompatible"
- github.com/go-redis/redis/v8: "v8.11.5"
- github.com/google/uuid: "v1.3.1"
- github.com/jmespath/go-jmespath: "v0.4.0" // indirect
- github.com/robfig/cron: "v1.2.0"
- github.com/streadway/amqp: "v1.1.0"
- golang.org/x/crypto: "v0.12.0"
- golang.org/x/net: "v0.14.0" // indirect
