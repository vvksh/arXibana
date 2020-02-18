FROM golang:latest
# RUN mkdir /app 
RUN apt-get update && apt-get install -y cron

WORKDIR /go/src/arxivProcessing/

COPY . .
RUN go mod download
RUN go build
RUN touch /var/log/cron.log
RUN echo "* * * * * /go/src/arxivProcessing/arxivProcessing >> /var/log/cron.log 2>&1" | crontab -  
CMD ./arxivProcessing -seed && cron && tail -f /var/log/cron.log
