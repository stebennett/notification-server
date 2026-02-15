FROM rabbitmq:3.13-management-alpine

# Download and install the delayed message exchange plugin
ADD --chmod=644 https://github.com/rabbitmq/rabbitmq-delayed-message-exchange/releases/download/v3.13.0/rabbitmq_delayed_message_exchange-3.13.0.ez /opt/rabbitmq/plugins/

# Enable plugins
RUN rabbitmq-plugins enable --offline rabbitmq_management rabbitmq_delayed_message_exchange
