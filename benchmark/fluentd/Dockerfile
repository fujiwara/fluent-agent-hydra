FROM alpine:latest

RUN apk --update add build-base ca-certificates ruby-dev ruby ruby-irb \
     && rm -rf /var/cache/apk/*

RUN echo 'gem: --no-document' >> /etc/gemrc
RUN gem install fluentd
RUN gem install fluent-plugin-flowcounter-simple

RUN mkdir -p /etc/fluentd
COPY fluentd.conf /etc/fluentd
EXPOSE 24224

CMD exec fluentd -c /etc/fluentd/fluentd.conf --no-supervisor
