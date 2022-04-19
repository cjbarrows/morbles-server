# Yermarbles server

## TODO

[ ] handle getAuthenticatedPlayer failing (ie, user not logged in yet)

## DONE

[x] return authenticated player
[x] add new user to player "table"
[x] clean up test code
[x] track failures
[x] persist players and levels in Postgres, Redis, or other store
    - Postgres
[ ] added isOfficial to Level and LevelStatus

## ENVIRONMENT VARIABLES

- CLIENT_DOMAIN   ".herokuapp.com" or ""

## LAUNCHING

go run .
