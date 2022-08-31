# This is a multi-stage Dockerfile and requires >= Docker 17.05
# https://docs.docker.com/engine/userguide/eng-image/multistage-build/
FROM gobuffalo/buffalo:v0.16.26 as builder

WORKDIR /src

ADD . .
RUN go get ./...
#RUN export GIT_COMMIT=$(git rev-list -1 HEAD) && \
#    buffalo build --ldflags "-X main.GitCommitHash=$GIT_COMMIT" --static -o /bin/app
RUN buffalo build --static -o /bin/app

FROM alpine
RUN apk add --no-cache bash ca-certificates

WORKDIR /bin/

COPY --from=builder /bin/app .

# Uncomment to run the binary in "production" mode:
ENV GO_ENV=production

# Bind the app to 0.0.0.0 so it can be seen from outside the container
ENV ADDR=0.0.0.0

EXPOSE 3000

CMD /bin/app migrate; /bin/app
