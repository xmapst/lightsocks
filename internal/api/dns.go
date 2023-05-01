package api

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"github.com/samber/lo"
	"github.com/xmapst/lightsocks/internal/resolver"
	"math"
	"net/http"
)

func queryDNS(c *gin.Context) {
	if resolver.DefaultResolver == nil {
		c.JSON(http.StatusInternalServerError, newError("DNS section is disabled"))
		return
	}

	name := c.Query("name")
	qTypeStr, _ := lo.Coalesce(c.Query("type"), "A")

	qType, exist := dns.StringToType[qTypeStr]
	if !exist {
		c.JSON(http.StatusBadRequest, newError("DNS section is disabled"))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
	defer cancel()

	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(name), qType)
	resp, err := resolver.DefaultResolver.ExchangeContext(ctx, &msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newError(err.Error()))
		return
	}

	responseData := gin.H{
		"Status":   resp.Rcode,
		"Question": resp.Question,
		"TC":       resp.Truncated,
		"RD":       resp.RecursionDesired,
		"RA":       resp.RecursionAvailable,
		"AD":       resp.AuthenticatedData,
		"CD":       resp.CheckingDisabled,
	}

	rr2Json := func(rr dns.RR, _ int) gin.H {
		header := rr.Header()
		return gin.H{
			"Name": header.Name,
			"Type": header.Rrtype,
			"TTL":  header.Ttl,
			"Data": lo.Substring(rr.String(), len(header.String()), math.MaxUint),
		}
	}

	if len(resp.Answer) > 0 {
		responseData["Answer"] = lo.Map(resp.Answer, rr2Json)
	}
	if len(resp.Ns) > 0 {
		responseData["Authority"] = lo.Map(resp.Ns, rr2Json)
	}
	if len(resp.Extra) > 0 {
		responseData["Additional"] = lo.Map(resp.Extra, rr2Json)
	}
	c.JSON(http.StatusOK, responseData)
}
