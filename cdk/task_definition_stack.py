from aws_cdk import (
    aws_ecs as ecs,
    aws_logs as logs,
    aws_iam as iam,
    aws_dynamodb as dynamodb,
    aws_ssm as ssm,
    aws_ecr as ecr,
    Stack,
    RemovalPolicy,
    Tags
)
from constructs import Construct

MC_VOLUME_NAME = 'mc-volume'

MULTICARRIER_EMAIL_ID = 'multicarrier-email'


class TaskDefinitionStack(Stack):
    def __init__(
            self,
            scope: Construct,
            id: str,
            env_parameters: dict,
            image_tag: str,
            **kwargs
    ) -> None:
        super().__init__(
            scope,
            id,
            **kwargs
        )

        service_name = env_parameters['SERVICE_NAME']
        selected_environment = env_parameters['SELECTED_ENVIRONMENT']
        mc_email_efs_folder_name = env_parameters['MC_EMAIL_EFS_FOLDER_NAME']
        service_cpu = env_parameters['SERVICE_CPU']
        service_memory = env_parameters['SERVICE_MEMORY']
        service_container_port = env_parameters['SERVICE_CONTAINER_PORT']
        service_host_port = env_parameters['SERVICE_HOST_PORT']

        outbox_table_name_parameter_name = env_parameters['OUTBOX_TABLE_NAME_PARAMETER_NAME']
        mc_eml_efs_access_point_arn_parameter_name = env_parameters['MC_EML_EFS_ACCESS_POINT_ARN_PARAMETER_NAME']
        mc_eml_efs_access_point_id_parameter_name = env_parameters['MC_EML_EFS_ACCESS_POINT_ID_PARAMETER_NAME']
        mc_eml_efs_id_parameter_name = env_parameters['MC_EML_EFS_ID_PARAMETER_NAME']
        repository_name_parameter_name = env_parameters['REPOSITORY_NAME_PARAMETER_NAME']
        tmp_task_definition_arn_parameter_name = env_parameters['TMP_TASK_DEFINITION_ARN_PARAMETER_NAME']
        ses_smtp_credentials_secret_name = env_parameters['SES_SMTP_CREDENTIALS_SECRET_NAME']
        callback_endpoint_parameter_name = env_parameters['CALLBACK_ENDPOINT_PARAMETER_NAME']
        smtp_user = env_parameters['SMTP_USER']
        smtp_password = env_parameters['SMTP_PASSWORD']
        smtp_sender = env_parameters['SMTP_SENDER']

        task_definition_family = f'{selected_environment}-{service_name}'

        task_definition = ecs.FargateTaskDefinition(
            scope=self,
            id=f'{service_name}-task-definition',
            cpu=int(service_cpu),
            family=task_definition_family,
            memory_limit_mib=int(service_memory)
        )

        task_definition.apply_removal_policy(
            policy=RemovalPolicy.RETAIN_ON_UPDATE_OR_DELETE
        )

        mc_eml_access_point_arn = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=mc_eml_efs_access_point_arn_parameter_name,
        )

        mc_eml_access_point_id = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=mc_eml_efs_access_point_id_parameter_name,
        )

        mc_email_efs_id = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=mc_eml_efs_id_parameter_name,
        )

        task_definition.add_to_execution_role_policy(
            statement=iam.PolicyStatement(
                actions=[
                    'elasticfilesystem:ClientMount',
                    'elasticfilesystem:ClientWrite',
                    'elasticfilesystem:ClientRootAccess'
                ],
                resources=[
                    mc_eml_access_point_arn
                ]
            )
        )

        repository_name = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=repository_name_parameter_name,
        )

        repository = ecr.Repository.from_repository_name(
            scope=self,
            id='ecr-repository',
            repository_name=repository_name,
        )

        task_definition.add_to_execution_role_policy(
            statement=iam.PolicyStatement(
                actions=[
                    'ecr:BatchCheckLayerAvailability',
                    'ecr:BatchGetImage',
                    'ecr:GetDownloadUrlForLayer'
                ],
                resources=[
                    repository.repository_arn
                ]
            )
        )

        log_group_retainment = RemovalPolicy.RETAIN if selected_environment == 'prod' else RemovalPolicy.DESTROY

        log_group = logs.LogGroup(
            scope=self,
            id=f'{service_name}-log-group',
            log_group_name=f'{selected_environment}/{MULTICARRIER_EMAIL_ID}/{service_name}',
            removal_policy=log_group_retainment,
            retention=logs.RetentionDays.ONE_MONTH
        )

        task_definition.add_to_execution_role_policy(
            statement=iam.PolicyStatement(
                actions=[
                    'logs:CreateLogStream',
                    'logs:PutLogEvents'
                ],
                resources=[
                    log_group.log_group_arn
                ]
            )
        )

        container = task_definition.add_container(
            id='container',
            image=ecs.ContainerImage.from_ecr_repository(
                repository=repository,
                tag=image_tag
            ),
            logging=ecs.LogDriver.aws_logs(
                stream_prefix=f'{selected_environment}/{service_name}',
                log_group=log_group
            )
        )

        container.add_port_mappings(
            ecs.PortMapping(
                container_port=int(service_container_port),
                host_port=int(service_host_port)
            )
        )

        task_definition.add_volume(
            name=MC_VOLUME_NAME,
            efs_volume_configuration=ecs.EfsVolumeConfiguration(
                file_system_id=mc_email_efs_id,
                transit_encryption='ENABLED',
                authorization_config=ecs.AuthorizationConfig(
                    access_point_id=mc_eml_access_point_id,
                    iam='ENABLED'
                )
            )
        )

        container.add_mount_points(
            ecs.MountPoint(
                container_path=mc_email_efs_folder_name,
                source_volume=MC_VOLUME_NAME,
                read_only=True
            )
        )

        table_name = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=outbox_table_name_parameter_name
        )

        table = dynamodb.Table.from_table_name(
            scope=self,
            id='table',
            table_name=table_name
        )

        table.grant_read_write_data(
            grantee=task_definition.task_role
        )

        table.grant(
            task_definition.task_role,
            'dynamodb:PartiQLSelect',
            'dynamodb:PartiQLInsert',
            'dynamodb:PartiQLUpdate',
            'dynamodb:PartiQLDelete'
        )

        task_definition.add_to_task_role_policy(
            statement=iam.PolicyStatement(
                actions=[
                    'dynamodb:PartiQLSelect'
                ],
                resources=[
                    f'{table.table_arn}/index/*'
                ]
            )
        )

        container.add_environment(
            name='EMAIL_OUTBOX_TABLE',
            value=table.table_name
        )

        callback_endpoint = ssm.StringParameter.value_from_lookup(
            scope=self,
            parameter_name=callback_endpoint_parameter_name
        )

        container.add_environment(
            name='PIPELINE_CALLBACK_URL',
            value=f'https://{callback_endpoint}/status-updates'
        )

        container.add_environment(
            name='PIPELINE_INTERVAL',
            value='5'
        )

        container.add_environment(
            name='SMTP_HOST',
            value='email-smtp.eu-west-1.amazonaws.com'
        )

        container.add_environment(
            name='SMTP_PORT',
            value='587'
        )

        container.add_environment(
            name='SMTP_USER',
            value=smtp_user
        )

        container.add_environment(
            name='SMTP_PASS',
            value=smtp_password
        )

        container.add_environment(
            name='SMTP_FROM',
            value=smtp_sender
        )

        container.add_environment(
            name='SMTP_CREDENTIALS_SECRET_NAME',
            value=ses_smtp_credentials_secret_name
        )

        ssm.StringParameter(
            scope=self,
            id='temporary-task-definition-arn',
            string_value=task_definition.task_definition_arn,
            parameter_name=tmp_task_definition_arn_parameter_name
        )

        Tags.of(task_definition).add('ecs_container_name', service_name)
        Tags.of(task_definition).add('task_family', task_definition_family)
        Tags.of(task_definition).add('image_tag', image_tag)
