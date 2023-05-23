# Overview of Internals

- utilizing a repository design, where the data that makes up the application is prioritized and each component (user, dono, etc,) is broken up and given its own repo for types and methods

for now, there is...
- users.go
- dono.go
- billing.go
- sol.go
- xmr.go
- utils.go

The interfaces on each file will show the methods associated with each repository.
