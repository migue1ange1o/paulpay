# Overview of Internals

Utilizing a repository design, where the data that makes up the application is prioritized and each component (user, dono, etc,) is broken up and given its own repo for types and methods

For now, there is...
- users.go
- dono.go
- billing.go
- sol.go
- xmr.go
- invite.go
- utils.go

Interfaces are provided on each page to show the methods that belong to that component's repository.

ToDo:
- Break up HTTP handlers from main.go
- Write tests for each data model and its functionality