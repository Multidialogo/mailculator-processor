#!/usr/bin/env python3
from aws_cdk import (
    App,
    Environment,
    Tags
)

from os import environ

from get_env_variables import GetEnvVariables
from task_definition_stack import TaskDefinitionStack

from multidialogo_cdk_shared.environment_secrets_resolver import EnvironmentSecretsResolver
from multidialogo_cdk_shared.enums import EnvironmentsEnum

if __name__ == "__main__":
    app = App()

    selected_environment = app.node.try_get_context('environment')
    image_tag = app.node.try_get_context('image_tag')
    dd_api_key_secret_name = app.node.try_get_context('dd_api_key_secret_name')
    smtp_user = app.node.try_get_context('smtp_user')
    smtp_password = app.node.try_get_context('smtp_password')

    env_parameters = GetEnvVariables(selected_environment).env_dict

    account = environ.get('CDK_DEFAULT_ACCOUNT')
    region = environ.get('CDK_DEFAULT_REGION')

    environment = Environment(account=account, region=region)

    environment_secrets_resolver = EnvironmentSecretsResolver(
        selected_environment=EnvironmentsEnum[selected_environment.upper()]
    )

    TaskDefinitionStack(
        app,
        f"{env_parameters['SELECTED_ENVIRONMENT']}-multicarrier-email-daemon-task-definition-stack",
        env_parameters=env_parameters,
        image_tag=image_tag,
        env=environment,
        environment_secrets_resolver=environment_secrets_resolver,
        smtp_user=smtp_user,
        smtp_password=smtp_password
    )

    Tags.of(app).add('env', selected_environment)
    Tags.of(app).add('ecs_cluster_name', selected_environment)

    app.synth()