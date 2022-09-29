
## 0.3.0 (Sep 29, 2022)

FEATURES:
* Compatibility with waypoint 0.10.x

IMPROVEMENTS:

BUG FIXES:

BREAKING CHANGES:
* Follow snake_case naming convention for hcl config keys (`StopOldInstances` --> `stop_old_instances`)


## 0.2.0 (Jun 22, 2022)

FEATURES:
* Compatibility with waypoint 0.8.x
* It's possible to specify application health check

IMPROVEMENTS:
* It's possible to stop all old instances after release (useful for apps using queues, where un-mapping the route doesn't stop the app from processing requests)
