/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package in10n

import "errors"

const quotaExceededPrefix = "quota exceeded: number of "

var ErrQuotaExceeded_Channels = errors.New(quotaExceededPrefix + "channels")
var ErrQuotaExceeded_ChannelsPerSubject = errors.New(quotaExceededPrefix + "channels per subject")
var ErrQuotaExceeded_Subsciptions = errors.New(quotaExceededPrefix + "subsciptions")
var ErrQuotaExceeded_SubsciptionsPerSubject = errors.New(quotaExceededPrefix + "subsciptions per subject")

var ErrChannelDoesNotExist = errors.New("channel does not exist")
