FROM golang:latest
# RUN mkdir /app 
RUN apt-get update && apt-get install -y cron

WORKDIR /go/src/arxivProcessing/

COPY . .
RUN go mod download
RUN go build
RUN touch /var/log/cron.log
RUN echo "0 0 * * * /go/src/arxivProcessing/arxivProcessing -search_query='cat:cs.DB+OR+cat:cs.DC' -index_name='dbanddistcomputing'>> /var/log/cron.log 2>&1" | crontab -
CMD ./arxivProcessing -seed -search_query="cat:cs.DB+OR+cat:cs.DC" -index_name="dbanddistcomputing" && cron && tail -f /var/log/cron.log
