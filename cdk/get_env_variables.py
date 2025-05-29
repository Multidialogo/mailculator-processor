import os

ENVIRONMENT_VARIABLES = [
    'AWS_REGION',
    'ACCOUNT_ID',
    'SERVICE_NAME',
    'MC_EMAIL_EFS_FOLDER_NAME',
    'SERVICE_CPU',
    'SERVICE_MEMORY',
    'SERVICE_CONTAINER_PORT',
    'SERVICE_HOST_PORT',
    'OUTBOX_TABLE_NAME_PARAMETER_NAME',
    'MC_EML_EFS_ACCESS_POINT_ARN_PARAMETER_NAME',
    'MC_EML_EFS_ACCESS_POINT_ID_PARAMETER_NAME',
    'MC_EML_EFS_ID_PARAMETER_NAME',
    'REPOSITORY_NAME_PARAMETER_NAME',
    'TMP_TASK_DEFINITION_ARN_PARAMETER_NAME',
    'CALLBACK_ENDPOINT_PARAMETER_NAME',
    'SES_SMTP_CREDENTIALS_SECRET_NAME',
    'SMTP_USER',
    'SMTP_PASSWORD',
    'SMTP_SENDER'
]

class GetEnvVariables:
    def __init__(
            self,
            selected_environment: str
    ) -> None:

        self.env_dict = {
            'SELECTED_ENVIRONMENT': selected_environment,
        }

        for i in ENVIRONMENT_VARIABLES:
            i_env = f'{selected_environment.upper()}_{i}'
            self.env_dict[i] = os.environ[i_env]
