import boto3

# Prompt the user for the region in which to create the VPC
region = input("Enter the AWS region to use (default: eu-west-1): ") or 'eu-west-1'

# Specify the base name, CIDR block, and tenancy of the VPC to create
base_name = 'vpc_image'
cidr_block = '10.0.0.0/24'
tenancy = 'default'

# Create a Boto3 EC2 client for the specified region
ec2 = boto3.client('ec2', region_name=region)

# Check if a VPC with the base name already exists
existing_vpcs = ec2.describe_vpcs(Filters=[{'Name': 'tag:Name', 'Values': [base_name]}])['Vpcs']

if existing_vpcs:
    # If a VPC with the base name exists, increment the name until a valid, unused name is found
    i = 1
    while True:
        new_name = f"{base_name}_{i}"
        existing_vpcs = ec2.describe_vpcs(Filters=[{'Name': 'tag:Name', 'Values': [new_name]}])['Vpcs']
        if not existing_vpcs:
            break
        i += 1
    vpc_name = new_name
else:
    # If no VPCs with the base name exist, use the base name
    vpc_name = base_name

# Create the VPC with the specified attributes
response = ec2.create_vpc(CidrBlock=cidr_block, InstanceTenancy=tenancy)

# Tag the VPC with the specified name
vpc_id = response['Vpc']['VpcId']
ec2.create_tags(Resources=[vpc_id], Tags=[{'Key': 'Name', 'Value': vpc_name}])

# Create an internet gateway and attach it to the VPC
igw_response = ec2.create_internet_gateway()
igw_id = igw_response['InternetGateway']['InternetGatewayId']
ec2.attach_internet_gateway(InternetGatewayId=igw_id, VpcId=vpc_id)
ec2.create_tags(Resources=[igw_id], Tags=[{'Key': 'Name', 'Value': f"{vpc_name}-igw"}])

# Create a subnet in the range of the VPC CIDR block
subnet_response = ec2.create_subnet(VpcId=vpc_id, CidrBlock=cidr_block)
subnet_id = subnet_response['Subnet']['SubnetId']
ec2.create_tags(Resources=[subnet_id], Tags=[{'Key': 'Name', 'Value': f"{vpc_name}-subnet"}])

# Associate the internet gateway with the default VPC route table and the new subnet
route_table_response = ec2.describe_route_tables(Filters=[{'Name': 'vpc-id', 'Values': [vpc_id]}])
route_table_id = route_table_response['RouteTables'][0]['RouteTableId']
ec2.create_route(RouteTableId=route_table_id, DestinationCidrBlock='0.0.0.0/0', GatewayId=igw_id)
ec2.associate_route_table(RouteTableId=route_table_id, SubnetId=subnet_id)
ec2.create_tags(Resources=[route_table_id], Tags=[{'Key': 'Name', 'Value': f"{vpc_name}-rt"}])

# Create a security group allowing SSH access to all IP addresses
sg_response = ec2.create_security_group(GroupName=f"{vpc_name}-sg", Description="SSH access from anywhere", VpcId=vpc_id)
sg_id = sg_response['GroupId']
ec2.authorize_security_group_ingress(GroupId=sg_id, IpPermissions=[{'IpProtocol': 'tcp', 'FromPort': 22, 'ToPort': 22, 'IpRanges': [{'CidrIp': '0.0.0.0/0'}]}])
ec2.create_tags(Resources=[sg_id], Tags=[{'Key': 'Name', 'Value': f"{vpc_name}-sg"}])

print(f"VPC '{vpc_name}' created with ID '{vpc_id}' in region '{region}'.")
print(f"Internet gateway created with ID '{igw_id}' and attached to VPC '{vpc_name}'.")
print(f"Subnet created with ID '{subnet_id}' in VPC '{vpc_name}'.")
print(f"Route table '{route_table_id}' associated with subnet '{subnet_id}'.")
print(f"Security group '{sg_id}' created for VPC '{vpc_name}'.")
