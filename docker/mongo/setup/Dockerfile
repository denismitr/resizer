FROM mongo:4.2

WORKDIR /usr/src/configs

COPY replicaSet.js .
COPY setup.sh .

RUN chmod +x ./setup.sh

CMD ["./setup.sh"]