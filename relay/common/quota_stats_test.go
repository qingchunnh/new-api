package common

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

// 渠道开启「不计入额度统计」时，StatsQuota 返回 0（用户总消耗与看板额度不再累加）；
// 其余情况原样返回额度。ChannelMeta 为 nil 时不能 panic（重试等流程会出现该状态）。
func TestRelayInfoStatsQuotaFollowsExcludeQuotaStats(t *testing.T) {
	info := &RelayInfo{ChannelMeta: &ChannelMeta{
		ChannelSetting: dto.ChannelSettings{ExcludeQuotaStats: true},
	}}
	require.True(t, info.ExcludesQuotaStats())
	require.Equal(t, 0, info.StatsQuota(500))

	normal := &RelayInfo{ChannelMeta: &ChannelMeta{}}
	require.False(t, normal.ExcludesQuotaStats())
	require.Equal(t, 500, normal.StatsQuota(500))

	noMeta := &RelayInfo{}
	require.False(t, noMeta.ExcludesQuotaStats())
	require.Equal(t, 500, noMeta.StatsQuota(500))
}
