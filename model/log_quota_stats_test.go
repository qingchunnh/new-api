package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 渠道开启「不计入额度统计」(ExcludeQuotaStats) 时：
// 日志明细保留真实额度（可查每笔花费），看板数据(quota_data)的额度写 0，
// 请求计数与 token 用量照常统计。
func TestRecordConsumeLogExcludeQuotaStatsOnlyZeroesQuotaDataQuota(t *testing.T) {
	originalDataExport := common.DataExportEnabled
	common.DataExportEnabled = true
	t.Cleanup(func() {
		common.DataExportEnabled = originalDataExport
	})

	cases := []struct {
		name               string
		excludeQuotaStats  bool
		wantQuotaDataQuota int
	}{
		{"excluded channel zeroes quota_data quota only", true, 0},
		{"normal channel keeps quota_data quota", false, 500},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			truncateTables(t)
			CacheQuotaDataLock.Lock()
			CacheQuotaData = make(map[string]*QuotaData)
			CacheQuotaDataLock.Unlock()

			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)

			RecordConsumeLog(c, 1, RecordConsumeLogParams{
				ChannelId:         7,
				PromptTokens:      10,
				CompletionTokens:  20,
				ModelName:         "gpt-test",
				TokenName:         "sk-test",
				Quota:             500,
				Content:           "test",
				TokenId:           1,
				IsStream:          true,
				Group:             "default",
				ExcludeQuotaStats: tc.excludeQuotaStats,
			})
			SaveQuotaDataCache()

			var logRow Log
			require.NoError(t, DB.Where("user_id = ?", 1).First(&logRow).Error)
			require.Equal(t, 500, logRow.Quota, "consume log must keep the real quota")

			var quotaDataRows []QuotaData
			require.NoError(t, DB.Find(&quotaDataRows).Error)
			require.Len(t, quotaDataRows, 1)
			require.Equal(t, tc.wantQuotaDataQuota, quotaDataRows[0].Quota)
			require.Equal(t, 1, quotaDataRows[0].Count, "request count must still be counted")
			require.Equal(t, 30, quotaDataRows[0].TokenUsed, "token usage must still be counted")
			require.Equal(t, 7, quotaDataRows[0].ChannelID)
		})
	}
}

// 任务差额结算日志同样遵守该口径：消费类型日志明细保留真实额度，quota_data 额度写 0。
func TestRecordTaskBillingLogExcludeQuotaStatsZeroesQuotaDataQuota(t *testing.T) {
	originalDataExport := common.DataExportEnabled
	originalLogConsume := common.LogConsumeEnabled
	common.DataExportEnabled = true
	common.LogConsumeEnabled = true
	t.Cleanup(func() {
		common.DataExportEnabled = originalDataExport
		common.LogConsumeEnabled = originalLogConsume
	})

	truncateTables(t)
	CacheQuotaDataLock.Lock()
	CacheQuotaData = make(map[string]*QuotaData)
	CacheQuotaDataLock.Unlock()

	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:             1,
		LogType:            LogTypeConsume,
		Content:            "任务差额结算",
		ChannelId:          9,
		ModelName:          "video-test",
		Quota:              300,
		TokenId:            0,
		Group:              "default",
		ExcludeQuotaStats:  true,
	})
	SaveQuotaDataCache()

	var logRow Log
	require.NoError(t, DB.Where("user_id = ?", 1).First(&logRow).Error)
	require.Equal(t, LogTypeConsume, logRow.Type)
	require.Equal(t, 300, logRow.Quota, "task billing log must keep the real quota")

	var quotaDataRows []QuotaData
	require.NoError(t, DB.Find(&quotaDataRows).Error)
	require.Len(t, quotaDataRows, 1)
	require.Equal(t, 0, quotaDataRows[0].Quota)
	require.Equal(t, 1, quotaDataRows[0].Count)
}
