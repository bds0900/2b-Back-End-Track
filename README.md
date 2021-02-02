## Note
1. Record the view count and click count in minute. So at 59th second, the counts from 1 to 59 seconds are counted and updated.  

2. Mock store is in-memory store. It is implemented by **map**. The time stamp is used as key and the view count and click count are value.  

3. There are two tickers, one is update the logs at 59th second, and another is write the count from temp logs to mock store every 5 second.  

4. Rate limit default values are:
>```go
>options = config{
>    windowMs:   1 * 60 * 1000, // unit in millisecond
>    max:        5,
>    message:    "Too many request",
>    statusCode: 429,
>}
>```

## End point

### **/stats/**
This point will show user all the log. Rate limit is set for this end point, developer can set the time and max count.  
Ther response return type is Json



### **/view/**
This point will count the view count and click count.

