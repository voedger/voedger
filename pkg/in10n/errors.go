/*
* Copyright (c) 2021-present unTill Pro, Ltd.
* @author Maxim Geraskin
 */

package in10n

import "errors"

const quotaExceededPrefix = "quota exceeded: number of "

var ErrQuotaExceeded_Channels = errors.New(quotaExceededPrefix + "channels")
var ErrQuotaExceeded_ChannelsPerSubject = errors.New(quotaExceededPrefix + "channels per subject")
var ErrQuotaExceeded_Subscriptions = errors.New(quotaExceededPrefix + "subscriptions")
var ErrQuotaExceeded_SubscriptionsPerSubject = errors.New(quotaExceededPrefix + "subscriptions per subject")

var ErrChannelDoesNotExist = errors.New("channel does not exist")
var ErrChannelTerminated = errors.New("channel terminated")
var ErrChannelAlreadyBeingWatched = errors.New("channel is already being watched")
