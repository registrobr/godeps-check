# godeps-check

Are you trying to follow all the Godeps dependencies commits from your project to quickly
identify when you should update your vendoring package? You're problems are solved!

With this tool you will get the last 10 commit messages made from each dependency of your
project. With that you can check if there's was some critical bug fix or some fantastic
new feature that you could use.

### How to use it

```
% go get github.com/registrobr/godeps-check
% cd $GOPATH/src/my-favorite-project
% godeps-check
```

Our suggestion is to add the execution of this tool to your crontab once a day and send you
a notification e-mail.
