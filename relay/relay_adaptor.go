package relay

import (
	"github.com/55gY/new-api-lite/constant"
	"github.com/55gY/new-api-lite/relay/channel"
	"github.com/55gY/new-api-lite/relay/channel/ali"
	"github.com/55gY/new-api-lite/relay/channel/aws"
	"github.com/55gY/new-api-lite/relay/channel/baidu"
	"github.com/55gY/new-api-lite/relay/channel/baidu_v2"
	"github.com/55gY/new-api-lite/relay/channel/claude"
	"github.com/55gY/new-api-lite/relay/channel/cloudflare"
	"github.com/55gY/new-api-lite/relay/channel/codex"
	"github.com/55gY/new-api-lite/relay/channel/cohere"
	"github.com/55gY/new-api-lite/relay/channel/coze"
	"github.com/55gY/new-api-lite/relay/channel/deepseek"
	"github.com/55gY/new-api-lite/relay/channel/dify"
	"github.com/55gY/new-api-lite/relay/channel/gemini"
	"github.com/55gY/new-api-lite/relay/channel/jimeng"
	"github.com/55gY/new-api-lite/relay/channel/jina"
	"github.com/55gY/new-api-lite/relay/channel/minimax"
	"github.com/55gY/new-api-lite/relay/channel/mistral"
	"github.com/55gY/new-api-lite/relay/channel/mokaai"
	"github.com/55gY/new-api-lite/relay/channel/moonshot"
	"github.com/55gY/new-api-lite/relay/channel/ollama"
	"github.com/55gY/new-api-lite/relay/channel/openai"
	"github.com/55gY/new-api-lite/relay/channel/palm"
	"github.com/55gY/new-api-lite/relay/channel/perplexity"
	"github.com/55gY/new-api-lite/relay/channel/replicate"
	"github.com/55gY/new-api-lite/relay/channel/siliconflow"
	"github.com/55gY/new-api-lite/relay/channel/submodel"
	"github.com/55gY/new-api-lite/relay/channel/tencent"
	"github.com/55gY/new-api-lite/relay/channel/vertex"
	"github.com/55gY/new-api-lite/relay/channel/volcengine"
	"github.com/55gY/new-api-lite/relay/channel/xai"
	"github.com/55gY/new-api-lite/relay/channel/xunfei"
	"github.com/55gY/new-api-lite/relay/channel/zhipu"
	"github.com/55gY/new-api-lite/relay/channel/zhipu_4v"
)

func GetAdaptor(apiType int) channel.Adaptor {
	switch apiType {
	case constant.APITypeAli:
		return &ali.Adaptor{}
	case constant.APITypeAnthropic:
		return &claude.Adaptor{}
	case constant.APITypeBaidu:
		return &baidu.Adaptor{}
	case constant.APITypeGemini:
		return &gemini.Adaptor{}
	case constant.APITypeOpenAI:
		return &openai.Adaptor{}
	case constant.APITypePaLM:
		return &palm.Adaptor{}
	case constant.APITypeTencent:
		return &tencent.Adaptor{}
	case constant.APITypeXunfei:
		return &xunfei.Adaptor{}
	case constant.APITypeZhipu:
		return &zhipu.Adaptor{}
	case constant.APITypeZhipuV4:
		return &zhipu_4v.Adaptor{}
	case constant.APITypeOllama:
		return &ollama.Adaptor{}
	case constant.APITypePerplexity:
		return &perplexity.Adaptor{}
	case constant.APITypeAws:
		return &aws.Adaptor{}
	case constant.APITypeCohere:
		return &cohere.Adaptor{}
	case constant.APITypeDify:
		return &dify.Adaptor{}
	case constant.APITypeJina:
		return &jina.Adaptor{}
	case constant.APITypeCloudflare:
		return &cloudflare.Adaptor{}
	case constant.APITypeSiliconFlow:
		return &siliconflow.Adaptor{}
	case constant.APITypeVertexAi:
		return &vertex.Adaptor{}
	case constant.APITypeMistral:
		return &mistral.Adaptor{}
	case constant.APITypeDeepSeek:
		return &deepseek.Adaptor{}
	case constant.APITypeMokaAI:
		return &mokaai.Adaptor{}
	case constant.APITypeVolcEngine:
		return &volcengine.Adaptor{}
	case constant.APITypeBaiduV2:
		return &baidu_v2.Adaptor{}
	case constant.APITypeOpenRouter:
		return &openai.Adaptor{}
	case constant.APITypeXinference:
		return &openai.Adaptor{}
	case constant.APITypeXai:
		return &xai.Adaptor{}
	case constant.APITypeCoze:
		return &coze.Adaptor{}
	case constant.APITypeJimeng:
		return &jimeng.Adaptor{}
	case constant.APITypeMoonshot:
		return &moonshot.Adaptor{} // Moonshot uses Claude API
	case constant.APITypeSubmodel:
		return &submodel.Adaptor{}
	case constant.APITypeMiniMax:
		return &minimax.Adaptor{}
	case constant.APITypeReplicate:
		return &replicate.Adaptor{}
	case constant.APITypeCodex:
		return &codex.Adaptor{}
	}
	return nil
}
