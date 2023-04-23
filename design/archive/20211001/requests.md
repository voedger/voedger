# Request URL

- `[<cluster>.<region>.]heeus.cloud/api/<region>/<zone>/<user>/<app>/<service>/<wsid>/<module>/<consistency>/<function>`
  - `<cluster>.<region>.` is an optional part, if missed request will be routed automatically
  - `<consistency>` = ( `l` | `h` )


- Built-in `mod` function is applied to module-mail
  - `heeus.cloud/api/ru/spb1/maxim/(app)mail/(service)main/(ws)890900/(module)mail/h/mod`
    - `mod` is a reserved function name
    - one-two-three-four-letters names are reserved
      - `sync`, `view`
    - Only `main` service may have high-consistency functions
- View partition log using built-in `heeus` module
  - `heeus.cloud/api/ru/spb1/maxim/mail/main/890900/heeus/l/log`