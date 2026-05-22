基于codex hook的快速模型转换

仿照 `/Users/jasonxu/Documents/personal/complexity-router` 设置router，但是去掉groq和任何自动switch的代码

注意!!!

使用codex hook
`https://developers.openai.com/codex/hooks`
不要自己造轮子！！！

## 原理
UserPromptSubmit hook 读取用户 prompt。
如果命中关键词，hook 运行一个curl到router的后台切换模型。
hook 返回 decision: "block"，阻止这次 prompt 进入模型。
同时脚本自己发 macOS notification(不要响铃)、告知用户 "model switched to <model name>"
env里可以自定义keyword和其对应的模型

## Example

```codex prompt
/msm
```
Above means "Model Switch Medium“ => switch to gpt-5.5, thinking effort: medium

## Note for Fast Mode

This is how it signal fast mode
```
{
  "model": "gpt-5.4-mini",
  "messages": [{ "role": "user", "content": "Reply with only: pong" }],
  "service_tier": "fast",
  "max_completion_tokens": 20
}
```
