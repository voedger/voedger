# implement an integration test that will check setting alias to the email of a deactivated profile

- URL: https://untill.atlassian.net/browse/AIR-4675
- ID: AIR-4675
- State: In Progress
- Author: Denis Gribanov
- Assignees: Denis Gribanov
- Labels: none

## Why

Case:

* there are user Profile 1 and Profile 2
* deactivate Profile 2
* set login alias for Profile 1 to email of Profile 2

Not clear will setting login alias will work in this case

## What

implement an integration test that will show the actual behaviour

* create 2 profiles
* create a child workspace in each
* fill the same field in the same field with different values

after setting the login alias check that under new alias we do not see values of a deactivated profile
