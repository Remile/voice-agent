# AWS 配置详细指南

本文档详细说明如何为语音对话系统配置 AWS 权限。

## 一、创建 IAM 用户（如果还没有）

### 1. 登录 AWS Console

访问 [AWS IAM Console](https://console.aws.amazon.com/iam/)

### 2. 创建新用户

1. 点击左侧 **Users**
2. 点击 **Add users**
3. 输入用户名，例如：`bedrock-voice-agent`
4. 访问类型选择：**Programmatic access**
5. 点击 **Next: Permissions**

### 3. 添加权限

#### 方式 1：使用托管策略（推荐用于测试）

在 "Attach existing policies directly" 中搜索并选择：
- `AmazonBedrockFullAccess`

#### 方式 2：创建自定义策略（推荐用于生产）

点击 **Create policy**，使用以下 JSON：

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "BedrockInvokeModel",
            "Effect": "Allow",
            "Action": [
                "bedrock:InvokeModel",
                "bedrock:InvokeModelWithResponseStream"
            ],
            "Resource": [
                "arn:aws:bedrock:us-east-1::foundation-model/us.amazon.nova-pro-v1:0",
                "arn:aws:bedrock:us-west-2::foundation-model/us.amazon.nova-pro-v1:0"
            ]
        }
    ]
}
```

命名策略为 `BedrockNovaInvokePolicy`，然后将其附加到用户。

### 4. 获取访问密钥

1. 点击 **Next: Tags**（可选）
2. 点击 **Next: Review**
3. 点击 **Create user**
4. **重要**：保存显示的 Access Key ID 和 Secret Access Key

## 二、启用 Bedrock Nova 模型访问

### 1. 切换到支持 Nova 的区域

在 AWS Console 右上角切换到：
- **US East (N. Virginia)** - us-east-1，或
- **US West (Oregon)** - us-west-2

### 2. 访问 Bedrock 服务

搜索并进入 **Amazon Bedrock** 服务

### 3. 请求模型访问

1. 点击左侧菜单 **Model access**
2. 点击右上角 **Manage model access** 或 **Request access**
3. 找到 **Amazon Nova** 部分
4. 勾选以下模型：
   - ✅ **Amazon Nova Pro**
   - ✅ **Amazon Nova Lite**（可选）
5. 勾选同意使用条款
6. 点击 **Request model access** 或 **Submit**

### 4. 等待审批

- 通常在几分钟内完成
- 状态从 "Access requested" 变为 "Access granted"
- 刷新页面查看最新状态

## 三、配置本地 AWS 凭证

### 方法 1：使用 AWS CLI（推荐）

#### 安装 AWS CLI

**macOS:**
```bash
brew install awscli
```

**其他系统:**
访问 [AWS CLI 安装指南](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)

#### 配置凭证

```bash
aws configure
```

输入信息：
```
AWS Access Key ID [None]: AKIA... (你的密钥)
AWS Secret Access Key [None]: wJalrXUtn... (你的密钥)
Default region name [None]: us-east-1
Default output format [None]: json
```

#### 验证配置

```bash
aws sts get-caller-identity
```

应该看到类似输出：
```json
{
    "UserId": "AIDACKCEVSQ6C2EXAMPLE",
    "Account": "123456789012",
    "Arn": "arn:aws:iam::123456789012:user/bedrock-voice-agent"
}
```

### 方法 2：使用环境变量

创建 `.env` 文件（基于 `.env.example`）：

```bash
cp .env.example .env
```

编辑 `.env` 文件，填入你的凭证：

```bash
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=wJalrXUtn...
AWS_REGION=us-east-1
```

运行程序前加载环境变量：

```bash
export $(cat .env | grep -v '^#' | xargs)
./voice-agent
```

### 方法 3：使用 AWS SSO

如果你的组织使用 AWS SSO：

#### 配置 SSO

```bash
aws configure sso
```

按提示输入：
- SSO Start URL
- SSO Region
- 选择账号和角色
- CLI default region: us-east-1
- CLI profile name: voice-agent

#### 登录

```bash
aws sso login --profile voice-agent
```

#### 使用指定 Profile 运行

```bash
export AWS_PROFILE=voice-agent
./voice-agent
```

## 四、测试 Bedrock 访问

### 使用 AWS CLI 测试

```bash
aws bedrock list-foundation-models \
    --region us-east-1 \
    --query "modelSummaries[?contains(modelId, 'nova')].[modelId,modelName]" \
    --output table
```

应该看到 Nova 模型列表：
```
-----------------------------------------------------------------
|                     ListFoundationModels                       |
+---------------------------------------+-------------------------+
|  us.amazon.nova-pro-v1:0             |  Amazon Nova Pro        |
|  us.amazon.nova-lite-v1:0            |  Amazon Nova Lite       |
+---------------------------------------+-------------------------+
```

### 测试调用模型

```bash
aws bedrock-runtime invoke-model \
    --region us-east-1 \
    --model-id us.amazon.nova-pro-v1:0 \
    --body '{"messages":[{"role":"user","content":[{"text":"Hello"}]}],"inferenceConfig":{"temperature":0.7}}' \
    --cli-binary-format raw-in-base64-out \
    /tmp/response.json

cat /tmp/response.json
```

## 五、常见问题和解决方案

### 问题 1：AccessDeniedException

```
An error occurred (AccessDeniedException) when calling the InvokeModel operation
```

**原因**：IAM 用户没有调用 Bedrock 的权限

**解决**：
1. 检查 IAM 策略是否包含 `bedrock:InvokeModel` 权限
2. 确认策略的 Resource ARN 正确
3. 等待几分钟让权限生效

### 问题 2：ValidationException: Model not found

```
ValidationException: Model us.amazon.nova-pro-v1:0 not found
```

**原因**：当前区域不支持 Nova 或未启用模型访问

**解决**：
1. 确认使用 us-east-1 或 us-west-2 区域
2. 在 Bedrock Console 中检查模型访问状态
3. 确保模型 ID 正确：`us.amazon.nova-pro-v1:0`

### 问题 3：ThrottlingException

```
ThrottlingException: Rate exceeded
```

**原因**：请求频率超过限制

**解决**：
1. 在代码中添加重试逻辑
2. 申请提高配额：AWS Console → Service Quotas → Amazon Bedrock
3. 减少请求频率

### 问题 4：ExpiredToken

```
ExpiredToken: The security token included in the request is expired
```

**原因**：AWS 凭证过期

**解决**：
- 如使用临时凭证，重新运行 `aws sso login`
- 如使用长期凭证，检查密钥是否有效

## 六、安全最佳实践

### 1. 使用最小权限原则

不要给用户过多权限，只授予必要的 Bedrock 调用权限。

### 2. 定期轮换访问密钥

```bash
# 创建新密钥
aws iam create-access-key --user-name bedrock-voice-agent

# 删除旧密钥
aws iam delete-access-key --user-name bedrock-voice-agent --access-key-id OLD_KEY_ID
```

### 3. 启用 CloudTrail 日志

监控 Bedrock API 调用：

1. 进入 AWS CloudTrail Console
2. 创建 Trail
3. 记录所有 Bedrock 事件

### 4. 设置成本告警

1. 进入 AWS Billing Console
2. 设置 Budget
3. 当支出超过阈值时接收通知

### 5. 不要提交凭证到 Git

确保 `.env` 和 `.aws/credentials` 在 `.gitignore` 中：

```bash
# 检查
git status

# 如果不小心提交了，立即轮换密钥
aws iam delete-access-key --access-key-id LEAKED_KEY_ID
```

## 七、成本估算

### Nova Pro 定价（截至 2024 年）

**输入（每 1000 tokens）**：
- 文本：$0.0008
- 图像：根据尺寸计算
- 音频：$0.001 / 秒

**输出（每 1000 tokens）**：
- 文本：$0.0032
- 音频：$0.004 / 秒

### 使用本应用的成本估算

**假设**：
- 每次对话录音 5 秒
- Nova 回复 10 秒音频
- 每天 50 次对话

**每日成本**：
```
输入音频：50 × 5 秒 × $0.001 = $0.25
输出音频：50 × 10 秒 × $0.004 = $2.00
总计：约 $2.25 / 天
```

**每月成本**：约 $67.50

**省钱建议**：
- 使用 Nova Lite（成本更低）
- 减少对话轮数
- 缩短录音时长
- 设置 AWS Budget 告警

## 八、区域选择建议

### US East (N. Virginia) - us-east-1

**优点**：
- ✅ 最早支持新服务
- ✅ 通常成本最低
- ✅ 最稳定

**缺点**：
- ❌ 从中国访问延迟较高

### US West (Oregon) - us-west-2

**优点**：
- ✅ 从中国访问延迟较低
- ✅ 支持大部分新功能

**缺点**：
- ❌ 价格可能略高

### 推荐

- **开发测试**：us-east-1（成本优先）
- **生产环境**：us-west-2（中国用户延迟优先）

## 九、监控和告警

### CloudWatch 指标

监控 Bedrock 使用情况：

```bash
aws cloudwatch get-metric-statistics \
    --namespace AWS/Bedrock \
    --metric-name Invocations \
    --dimensions Name=ModelId,Value=us.amazon.nova-pro-v1:0 \
    --start-time 2024-01-01T00:00:00Z \
    --end-time 2024-01-02T00:00:00Z \
    --period 3600 \
    --statistics Sum
```

### 设置告警

1. 进入 CloudWatch Console
2. 创建 Alarm
3. 选择指标：Bedrock > ModelInvocations
4. 设置阈值（例如：每小时 > 100 次）
5. 配置 SNS 通知

## 十、故障排查清单

运行程序前检查：

- [ ] AWS 凭证已配置（`aws sts get-caller-identity`）
- [ ] 使用正确的区域（us-east-1 或 us-west-2）
- [ ] Nova Pro 模型访问已启用
- [ ] IAM 权限包含 `bedrock:InvokeModel`
- [ ] 网络可以访问 AWS API（如在中国，可能需要代理）
- [ ] 程序已编译（`go build`）
- [ ] 麦克风和扬声器正常工作

## 帮助和支持

- AWS Bedrock 文档：https://docs.aws.amazon.com/bedrock/
- AWS SDK for Go 文档：https://aws.github.io/aws-sdk-go-v2/
- 提交 Issue：在项目 GitHub 仓库创建 Issue

---

完成以上配置后，你就可以开始使用语音对话系统了！🎉

