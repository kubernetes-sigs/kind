# Scripts

# Create a markdown table for all scripts with name and a small brief description

| Script Name | Description |
|-------------|-------------|
| aws_images_vpc.py | Vpc creation for images building (if no default vpc exists) |

## aws_images_vpc.py

> What the script does?  
> This Python script uses the Boto3 library to create a Virtual Private Cloud (VPC) in Amazon Web Services (AWS). The script prompts the user for a VPC name and a region, and then creates a VPC with the specified name and CIDR block. It then creates an Internet Gateway, a subnet in the VPC, and a Security Group that allows SSH traffic. Finally, the script associates the Internet Gateway with the VPC's default route table and the new subnet.
 
> Boto3 is a Python library that allows you to interact with Amazon Web Services (AWS) services, such as EC2, S3, and many others, programmatically.  
To use Boto3, you need to provide AWS credentials that allow access to the services you want to use.
> Configuring AWS Credentials on Linux  
To configure your AWS credentials on a Linux machine, you can use the AWS CLI tool. Follow these steps:  
>> * Install the AWS CLI tool by running the following command in your terminal:  
>> `sudo apt install awscli`  
>> * Once installed, run the `aws configure` command to set up your credentials. This command will prompt you to enter your Access Key ID, Secret Access Key, default region name, and default output format.  
>> * Enter your Access Key ID and Secret Access Key when prompted. These keys can be generated in the AWS Management Console by navigating to the IAM service and creating a new user with programmatic access.  
>> * After entering your credentials, you will be prompted to enter the default region name. This is the region that the AWS CLI will use by default for any AWS service commands you run. You can set this to the region where you want to create your VPC.  
>> * Finally, you will be prompted to enter the default output format. This is the format in which the AWS CLI will display its output. You can choose either json, text, or table.  

> Once you have configured your credentials, you can use the boto3.Session() method in your Python script to create a session with AWS using your credentials. This will allow you to interact with AWS services programmatically using the Boto3 library.  
> Here is an example of how to create a session using the credentials configured with the AWS CLI:  
```python
import boto3

# Create a session using the default profile
session = boto3.Session(profile_name='default')

# Use the session to create an EC2 client
ec2 = session.client('ec2')
```
> In this example, we are creating a session using the default profile, which is the profile created by the aws configure command. We are then using the session to create an EC2 client, which can be used to create and manage EC2 instances in AWS.  
