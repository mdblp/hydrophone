# Development
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS development
ARG APP_VERSION
ENV APP_VERSION=${APP_VERSION}
ENV GO111MODULE=on

WORKDIR /go/src/github.com/tidepool-org/hydrophone
ARG GITHUB_TOKEN

COPY . .

RUN apk --no-cache update && \
    apk --no-cache upgrade && \
    apk add --no-cache gcc musl-dev git rsync

RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

ARG TARGETPLATFORM
ARG BUILDPLATFORM
RUN  ./build.sh $TARGETPLATFORM

CMD ["./dist/hydrophone"]

# Production
FROM gcr.io/distroless/static:nonroot AS production
WORKDIR /home/nonroot
USER nonroot
ENV GO111MODULE=on
COPY --from=development --chown=nonroot /go/src/github.com/tidepool-org/hydrophone/dist/hydrophone .
COPY --chown=nonroot templates/html ./templates/html/
COPY --chown=nonroot templates/locales ./templates/locales/
COPY --chown=nonroot templates/meta ./templates/meta/

CMD ["./hydrophone"]
