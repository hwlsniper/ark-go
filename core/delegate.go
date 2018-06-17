package core

import (
	"net/http"
	"strconv"
	"strings"
)

//DelegateResponse data - received from api-call.
type DelegateResponse struct {
	Success        bool           `json:"success,omitempty"`
	Delegates      []DelegateData `json:"delegates,omitempty"`
	SingleDelegate DelegateData   `json:"delegate,omitempty"`
	TotalCount     int            `json:"totalCount,omitempty"`
}

//DelegateVoters struct to hold voters for a publicKey(delegate)
type DelegateVoters struct {
	Success  bool `json:"success"`
	Accounts []struct {
		Username  string `json:"username"`
		Address   string `json:"address"`
		PublicKey string `json:"publicKey"`
		Balance   string `json:"balance"`
	} `json:"accounts"`
}

//DelegateData holds parsed json from api calls. It is used in upper DelegateResponse struct
type DelegateData struct {
	Username       string  `json:"username"`
	Address        string  `json:"address"`
	PublicKey      string  `json:"publicKey"`
	Vote           string  `json:"vote"`
	Producedblocks int     `json:"producedblocks"`
	Missedblocks   int     `json:"missedblocks"`
	Rate           int     `json:"rate"`
	Approval       float64 `json:"approval"`
	Productivity   float64 `json:"productivity"`
}

//ForgedDetails structure to hold delegate details
type ForgedDetails struct {
	Success bool   `json:"success"`
	Fees    string `json:"fees"`
	Rewards string `json:"rewards"`
	Forged  string `json:"forged"`
}

//DelegateQueryParams - when set, they are automatically added to get requests
type DelegateQueryParams struct {
	UserName  string `url:"username,omitempty"`
	PublicKey string `url:"publicKey,omitempty"`
	Offset    int    `url:"offset,omitempty"`
	OrderBy   string `url:"orderBy,omitempty"`
	Limit     int    `url:"limit,omitempty"`
}

//DelegateDataProfit structure - returns calculated profits based on input parameters
type DelegateDataProfit struct {
	Address         string
	VoteWeight      float64
	VoteWeightShare float64
	EarnedAmount100 float64 //100 earned amount.
	EarnedAmountXX  float64 //XX share to be payed
	VoteDuration    int     //Duration of vote in Hours
}

//ListDelegates function returns list of delegates. The top 51 delegates are returned
func (s *ArkClient) ListDelegates(params DelegateQueryParams) (DelegateResponse, *http.Response, error) {
	respData := new(DelegateResponse)
	respError := new(ArkApiResponseError)
	resp, err := s.sling.New().Get("api/delegates").QueryStruct(&params).Receive(respData, respError)
	if err == nil {
		err = respError
	}

	return *respData, resp, err
}

//GetForgedData details
func (s *ArkClient) GetForgedData(params DelegateQueryParams) (ForgedDetails, *http.Response, error) {
	respData := new(ForgedDetails)
	respError := new(ArkApiResponseError)

	qstr := "generatorPublicKey=" + params.PublicKey

	resp, err := s.sling.New().Get("api/delegates/forging/getForgedByAccount?"+qstr).Receive(respData, respError)
	if err == nil {
		err = respError
	}

	return *respData, resp, err
}

//GetDelegate function returns a delegate
func (s *ArkClient) GetDelegate(params DelegateQueryParams) (DelegateResponse, *http.Response, error) {
	respData := new(DelegateResponse)
	respError := new(ArkApiResponseError)
	resp, err := s.sling.New().Get("api/delegates/get").QueryStruct(&params).Receive(respData, respError)
	if err == nil {
		err = respError
	}

	return *respData, resp, err
}

//GetDelegateVoters function returns a delegates voters
func (s *ArkClient) GetDelegateVoters(params DelegateQueryParams) (DelegateVoters, *http.Response, error) {
	respData := new(DelegateVoters)
	respError := new(ArkApiResponseError)
	resp, err := s.sling.New().Get("api/delegates/voters").QueryStruct(&params).Receive(respData, respError)
	if err == nil {
		err = respError
	}

	return *respData, resp, err
}

//GetDelegateVoteWeight function returns a summary of ARK voted for selected delegate
func (s *ArkClient) GetDelegateVoteWeight(params DelegateQueryParams) (int, *http.Response, error) {
	respData := new(DelegateVoters)
	respError := new(ArkApiResponseError)
	resp, err := s.sling.New().Get("api/delegates/voters").QueryStruct(&params).Receive(respData, respError)
	if err == nil {
		err = respError
	}

	//calculating vote weight
	balance := 0
	if respData.Success {
		for _, element := range respData.Accounts {
			intBalance, _ := strconv.Atoi(element.Balance)
			balance += intBalance
		}
	}

	return balance, resp, err
}

func isBlockedAddress(list string, address string) bool {
	//blocklist checling and excluding
	if len(list) > 0 {
		return strings.Contains(strings.ToLower(list), strings.ToLower(address))
	}
	return false
}

func isAllowedAddress(list string, address string) bool {
	//blocklist checling and excluding
	if len(list) > 0 {
		return strings.Contains(strings.ToLower(list), strings.ToLower(address))
	}
	return false
}

//CalculateVotersProfit returns voter calculation details - based on settings
func (s *ArkClient) CalculateVotersProfit(params DelegateQueryParams, shareRatio float64, blocklist string, whitelist string, capBalance bool, balanceCapAmount float64, blockBalanceCap bool) []DelegateDataProfit {
	delegateRes, _, _ := s.GetDelegate(params)
	voters, _, _ := s.GetDelegateVoters(params)
	accountRes, _, _ := s.GetAccount(AccountQueryParams{Address: delegateRes.SingleDelegate.Address})

	delegateBalance, _ := strconv.ParseFloat(accountRes.Account.Balance, 64)
	delegateBalance = float64(delegateBalance) / SATOSHI

	//calculating vote weight
	votersProfit := []DelegateDataProfit{}
	delelgateVoteWeight := 0

	//computing summ of all votes
	for _, element := range voters.Accounts {
		//skipping blocked ones
		if isBlockedAddress(blocklist, element.Address) {
			continue
		}

		//skip balanceCap unless whitelisted
		currentVoterBalance, _ := strconv.ParseFloat(element.Balance, 64)
		intBalance, _ := strconv.Atoi(element.Balance)
		if capBalance && currentVoterBalance > balanceCapAmount {
			if !isAllowedAddress(whitelist, element.Address) {
				if blockBalanceCap {
					intBalance = 0
					continue
				} else {
					intBalance = int(balanceCapAmount)
				}
			}
		}
		delelgateVoteWeight += intBalance
	}

	//calculating
	for _, element := range voters.Accounts {
		//skipping blocked ones
		if isBlockedAddress(blocklist, element.Address) {
			continue
		}

		//skip balanceCap unless whitelisted
		currentVoterBalance, _ := strconv.ParseFloat(element.Balance, 64)
		if capBalance && currentVoterBalance > balanceCapAmount {
			if !isAllowedAddress(whitelist, element.Address) {
				if blockBalanceCap {
					currentVoterBalance = 0
					continue
				} else {
					currentVoterBalance = balanceCapAmount
				}
			}
		}

		deleProfit := DelegateDataProfit{
			Address: element.Address,
		}
		deleProfit.VoteWeight = currentVoterBalance / SATOSHI
		deleProfit.VoteWeightShare = float64(currentVoterBalance) / float64(delelgateVoteWeight)
		deleProfit.EarnedAmount100 = float64(delegateBalance) * deleProfit.VoteWeightShare
		deleProfit.EarnedAmountXX = float64(delegateBalance) * deleProfit.VoteWeightShare * shareRatio
		deleProfit.VoteDuration = s.GetVoteDuration(element.Address)
		votersProfit = append(votersProfit, deleProfit)
	}

	return votersProfit
}

//GetVoteDuration returns vote duration in HOURS
func (s *ArkClient) GetVoteDuration(address string) int {
	transQuery := TransactionQueryParams{SenderID: address, OrderBy: "timestamp:desc"}
	transResp, _, _ := s.ListTransaction(transQuery)

	duration := 0
	for _, element := range transResp.Transactions {
		if element.Type == VOTE {
			duration = GetDurationTime(element.Timestamp)
			break
		}
	}
	return duration
}
