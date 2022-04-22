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
    [x] Heroku not setting cookie (again?):
        - "This attempt to set a cookie via a Set-Cookie header was blocked because its Domain attribute was invalid with regards to the current host url."
        - maybe because it's "heroku.com" (which is on the list of "don't allow domain cookies")
        [-] try "morbles.herokuapp.com"
        [-] try "morbles-server.herokuapp.com"
              "This Set-Cookie header didn't specify a "SameSite" attribute and was defaulted to "SameSite=Lax," and was blocked because it came from a cross-site response which was not the response to a top-level navigation. The Set-Cookie had to have been set with "SameSite=None" to enable cross-site usage."
        [-] try "*.herokuapp.com"
        [-] try with SameSite=None
            "This attempt to set a cookie via a Set-Cookie header was blocked because it had the "SameSite=None" attribute but did not have the "Secure" attribute, which is required in order to use "SameSite=None".
        [x] try with "Secure"
            - now it's working -- but I wonder why it *stopped* working??
            - I thought for a while I'd put the client code on the server, but I guess I hadn't...
    [ ] switch from cookie session to redis session?
        - not needed... for now...

## ENVIRONMENT VARIABLES

- CLIENT_DOMAIN   ".herokuapp.com" or ""

## LAUNCHING

``go run .``
