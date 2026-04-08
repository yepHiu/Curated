# Jellyfin播放链路实现

Jellyfin的播放链路主要通过三种模式实现：DirectPlay（直接播放）、DirectStream（直接流）和Transcode（转码）。 [1](#0-0) 

## 播放链路流程

### 1. 请求入口
播放请求通过多个Controller处理：
- `DynamicHlsController` - 处理HLS流请求（/Videos/{id}/master.m3u8, /Videos/{id}/live.m3u8） [2](#0-1) 
- `VideosController` - 处理渐进式视频流（/Videos/{id}/stream） [3](#0-2) 
- `UniversalAudioController` - 处理通用音频流 [4](#0-3) 

### 2. 流状态构建
所有请求都会调用`StreamingHelpers.GetStreamingState`构建流状态： [1](#0-0) 
- 解析请求参数
- 获取媒体源信息
- 确定输出容器和编解码器
- 构建StreamState对象

### 3. 播放模式决策
根据客户端设备能力和媒体源特性选择播放模式：
- **DirectPlay**: 直接返回原始文件
- **DirectStream**: 仅重新封装容器（-c copy）
- **Transcode**: 完全重新编码

## 转码实现

### 1. 转码启动
当需要转码时，通过`TranscodeManager.StartFfMpeg`启动FFmpeg进程： [5](#0-4) 

```csharp
// 启动FFmpeg进程
var process = new Process
{
    StartInfo = new ProcessStartInfo
    {
        FileName = _mediaEncoder.EncoderPath,
        Arguments = commandLineArguments,
        // ...
    }
};
```

### 2. 转码类型判断
根据输出编解码器确定转码类型： [6](#0-5) 
- 完全转码：FFmpeg.Transcode-
- 重封装：FFmpeg.Remux-
- 直接流：FFmpeg.DirectStream-

### 3. 转码控制
- **节流控制**：通过`TranscodingThrottler`控制转码速度 [7](#0-6) 
- **分段清理**：通过`TranscodingSegmentCleaner`清理旧分段 [8](#0-7) 

## 推流实现

### 1. HLS流生成
HLS推流通过`DynamicHlsController`实现：
- 生成主播放列表（master.m3u8） [9](#0-8) 
- 生成变体播放列表（main.m3u8） [10](#0-9) 
- 提供分段文件服务 [11](#0-10) 

### 2. 播放列表构建
`DynamicHlsHelper`负责构建M3U8播放列表： [12](#0-11) 
- 添加EXTM3U头部
- 构建流信息
- 支持自适应码率

### 3. 分段处理
- 检查分段是否存在
- 如不存在则启动转码生成分段
- 支持分段缓存和按需生成 [13](#0-12) 

## 关键组件

1. **StreamState** - 包含所有流状态信息 [14](#0-13) 
2. **TranscodingJob** - 管理转码任务生命周期 [15](#0-14) 
3. **EncodingHelper** - 构建FFmpeg命令行参数

整个播放链路通过这些组件协同工作，实现了从媒体源到客户端的完整流媒体传输管道。

## Notes

- 播放链路支持实时流和点播流两种模式
- 转码过程可通过硬件加速提升性能
- HLS推流支持自适应码率和字幕轨道
- 所有转码任务都会记录日志以便调试

Wiki pages you might want to explore:
- [Media Streaming (jellyfin/jellyfin)](/wiki/jellyfin/jellyfin#3)

### Citations

**File:** Jellyfin.Api/Helpers/StreamingHelpers.cs (L47-259)
```csharp
    public static async Task<StreamState> GetStreamingState(
        StreamingRequestDto streamingRequest,
        HttpContext httpContext,
        IMediaSourceManager mediaSourceManager,
        IUserManager userManager,
        ILibraryManager libraryManager,
        IServerConfigurationManager serverConfigurationManager,
        IMediaEncoder mediaEncoder,
        EncodingHelper encodingHelper,
        ITranscodeManager transcodeManager,
        TranscodingJobType transcodingJobType,
        CancellationToken cancellationToken)
    {
        var httpRequest = httpContext.Request;
        if (!string.IsNullOrWhiteSpace(streamingRequest.Params))
        {
            ParseParams(streamingRequest);
        }

        streamingRequest.StreamOptions = ParseStreamOptions(httpRequest.Query);
        if (httpRequest.Path.Value is null)
        {
            throw new ResourceNotFoundException(nameof(httpRequest.Path));
        }

        var url = httpRequest.Path.Value.AsSpan().RightPart('.').ToString();

        if (string.IsNullOrEmpty(streamingRequest.AudioCodec))
        {
            streamingRequest.AudioCodec = encodingHelper.InferAudioCodec(url);
        }

        var state = new StreamState(mediaSourceManager, transcodingJobType, transcodeManager)
        {
            Request = streamingRequest,
            RequestedUrl = url,
            UserAgent = httpRequest.Headers[HeaderNames.UserAgent]
        };

        var userId = httpContext.User.GetUserId();
        if (!userId.IsEmpty())
        {
            state.User = userManager.GetUserById(userId);
        }

        if (state.IsVideoRequest && !string.IsNullOrWhiteSpace(state.Request.VideoCodec))
        {
            state.SupportedVideoCodecs = state.Request.VideoCodec.Split(',', StringSplitOptions.RemoveEmptyEntries);
            state.Request.VideoCodec = state.SupportedVideoCodecs.FirstOrDefault();
        }

        if (!string.IsNullOrWhiteSpace(streamingRequest.AudioCodec))
        {
            state.SupportedAudioCodecs = streamingRequest.AudioCodec.Split(',', StringSplitOptions.RemoveEmptyEntries);
            state.Request.AudioCodec = state.SupportedAudioCodecs.FirstOrDefault(mediaEncoder.CanEncodeToAudioCodec)
                                       ?? state.SupportedAudioCodecs.FirstOrDefault();
        }

        if (!string.IsNullOrWhiteSpace(streamingRequest.SubtitleCodec))
        {
            state.SupportedSubtitleCodecs = streamingRequest.SubtitleCodec.Split(',', StringSplitOptions.RemoveEmptyEntries);
            state.Request.SubtitleCodec = state.SupportedSubtitleCodecs.FirstOrDefault(mediaEncoder.CanEncodeToSubtitleCodec)
                                          ?? state.SupportedSubtitleCodecs.FirstOrDefault();
        }

        var item = libraryManager.GetItemById<BaseItem>(streamingRequest.Id)
            ?? throw new ResourceNotFoundException();

        state.IsInputVideo = item.MediaType == MediaType.Video;

        MediaSourceInfo? mediaSource = null;
        if (string.IsNullOrWhiteSpace(streamingRequest.LiveStreamId))
        {
            var currentJob = !string.IsNullOrWhiteSpace(streamingRequest.PlaySessionId)
                ? transcodeManager.GetTranscodingJob(streamingRequest.PlaySessionId)
                : null;

            if (currentJob is not null)
            {
                mediaSource = currentJob.MediaSource;
            }

            if (mediaSource is null)
            {
                var mediaSources = await mediaSourceManager.GetPlaybackMediaSources(libraryManager.GetItemById<BaseItem>(streamingRequest.Id), null, false, false, cancellationToken).ConfigureAwait(false);

                mediaSource = string.IsNullOrEmpty(streamingRequest.MediaSourceId)
                    ? mediaSources[0]
                    : mediaSources.FirstOrDefault(i => string.Equals(i.Id, streamingRequest.MediaSourceId, StringComparison.Ordinal));

                if (mediaSource is null && Guid.Parse(streamingRequest.MediaSourceId).Equals(streamingRequest.Id))
                {
                    mediaSource = mediaSources[0];
                }
            }
        }
        else
        {
            var liveStreamInfo = await mediaSourceManager.GetLiveStreamWithDirectStreamProvider(streamingRequest.LiveStreamId, cancellationToken).ConfigureAwait(false);
            mediaSource = liveStreamInfo.Item1;
            state.DirectStreamProvider = liveStreamInfo.Item2;

            // Cap the max bitrate when it is too high. This is usually due to ffmpeg is unable to probe the source liveTV streams' bitrate.
            if (mediaSource.FallbackMaxStreamingBitrate is not null && streamingRequest.VideoBitRate is not null)
            {
                streamingRequest.VideoBitRate = Math.Min(streamingRequest.VideoBitRate.Value, mediaSource.FallbackMaxStreamingBitrate.Value);
            }
        }

        var encodingOptions = serverConfigurationManager.GetEncodingOptions();

        encodingHelper.AttachMediaSourceInfo(state, encodingOptions, mediaSource, url);

        string? containerInternal = Path.GetExtension(state.RequestedUrl);

        if (string.IsNullOrEmpty(containerInternal)
            && (!string.IsNullOrWhiteSpace(streamingRequest.LiveStreamId)
                || (mediaSource != null && mediaSource.IsInfiniteStream)))
        {
            containerInternal = ".ts";
        }

        if (!string.IsNullOrEmpty(streamingRequest.Container))
        {
            containerInternal = streamingRequest.Container;
        }

        if (string.IsNullOrEmpty(containerInternal))
        {
            containerInternal = streamingRequest.Static ?
                StreamBuilder.NormalizeMediaSourceFormatIntoSingleContainer(state.InputContainer, null, DlnaProfileType.Audio)
                : GetOutputFileExtension(state, mediaSource);
        }

        var outputAudioCodec = streamingRequest.AudioCodec;
        state.OutputAudioCodec = outputAudioCodec;
        state.OutputContainer = (containerInternal ?? string.Empty).TrimStart('.');
        state.OutputAudioChannels = encodingHelper.GetNumAudioChannelsParam(state, state.AudioStream, state.OutputAudioCodec);
        if (EncodingHelper.LosslessAudioCodecs.Contains(outputAudioCodec))
        {
            state.OutputAudioBitrate = state.AudioStream.BitRate ?? 0;
        }
        else
        {
            state.OutputAudioBitrate = encodingHelper.GetAudioBitrateParam(streamingRequest.AudioBitRate, streamingRequest.AudioCodec, state.AudioStream, state.OutputAudioChannels) ?? 0;
        }

        if (outputAudioCodec.StartsWith("pcm_", StringComparison.Ordinal))
        {
            containerInternal = ".pcm";
        }

        if (state.VideoRequest is not null)
        {
            state.OutputVideoCodec = state.Request.VideoCodec;
            state.OutputVideoBitrate = encodingHelper.GetVideoBitrateParamValue(state.VideoRequest, state.VideoStream, state.OutputVideoCodec);

            encodingHelper.TryStreamCopy(state, encodingOptions);

            if (!EncodingHelper.IsCopyCodec(state.OutputVideoCodec) && state.OutputVideoBitrate.HasValue)
            {
                var isVideoResolutionNotRequested = !state.VideoRequest.Width.HasValue
                    && !state.VideoRequest.Height.HasValue
                    && !state.VideoRequest.MaxWidth.HasValue
                    && !state.VideoRequest.MaxHeight.HasValue;

                if (isVideoResolutionNotRequested
                    && state.VideoStream is not null
                    && state.VideoRequest.VideoBitRate.HasValue
                    && state.VideoStream.BitRate.HasValue
                    && state.VideoRequest.VideoBitRate.Value >= state.VideoStream.BitRate.Value)
                {
                    // Don't downscale the resolution if the width/height/MaxWidth/MaxHeight is not requested,
                    // and the requested video bitrate is greater than source video bitrate.
                    if (state.VideoStream.Width.HasValue || state.VideoStream.Height.HasValue)
                    {
                        state.VideoRequest.MaxWidth = state.VideoStream?.Width;
                        state.VideoRequest.MaxHeight = state.VideoStream?.Height;
                    }
                }
                else
                {
                    var h264EquivalentBitrate = EncodingHelper.ScaleBitrate(
                        state.OutputVideoBitrate.Value,
                        state.ActualOutputVideoCodec,
                        "h264");
                    var resolution = ResolutionNormalizer.Normalize(
                        state.VideoStream?.BitRate,
                        state.OutputVideoBitrate.Value,
                        h264EquivalentBitrate,
                        state.VideoRequest.MaxWidth,
                        state.VideoRequest.MaxHeight,
                        state.TargetFramerate);

                    state.VideoRequest.MaxWidth = resolution.MaxWidth;
                    state.VideoRequest.MaxHeight = resolution.MaxHeight;
                }
            }

            if (state.AudioStream is not null && !EncodingHelper.IsCopyCodec(state.OutputAudioCodec) && string.Equals(state.AudioStream.Codec, state.OutputAudioCodec, StringComparison.OrdinalIgnoreCase) && state.OutputAudioBitrate.HasValue)
            {
                state.OutputAudioCodec = state.SupportedAudioCodecs.Where(c => !EncodingHelper.LosslessAudioCodecs.Contains(c)).FirstOrDefault(mediaEncoder.CanEncodeToAudioCodec);
            }
        }

        var ext = string.IsNullOrWhiteSpace(state.OutputContainer)
            ? GetOutputFileExtension(state, mediaSource)
            : ("." + GetContainerFileExtension(state.OutputContainer));

        state.OutputFilePath = GetOutputFilePath(state, ext, serverConfigurationManager, streamingRequest.DeviceId, streamingRequest.PlaySessionId);

        return state;
    }
```

**File:** Jellyfin.Api/Controllers/DynamicHlsController.cs (L164-221)
```csharp
    [HttpGet("Videos/{itemId}/live.m3u8")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesPlaylistFile]
    public async Task<ActionResult> GetLiveHlsStream(
        [FromRoute, Required] Guid itemId,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? container,
        [FromQuery] bool? @static,
        [FromQuery] string? @params,
        [FromQuery] string? tag,
        [FromQuery, ParameterObsolete] string? deviceProfileId,
        [FromQuery] string? playSessionId,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? segmentContainer,
        [FromQuery] int? segmentLength,
        [FromQuery] int? minSegments,
        [FromQuery] string? mediaSourceId,
        [FromQuery] string? deviceId,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? audioCodec,
        [FromQuery] bool? enableAutoStreamCopy,
        [FromQuery] bool? allowVideoStreamCopy,
        [FromQuery] bool? allowAudioStreamCopy,
        [FromQuery] int? audioSampleRate,
        [FromQuery] int? maxAudioBitDepth,
        [FromQuery] int? audioBitRate,
        [FromQuery] int? audioChannels,
        [FromQuery] int? maxAudioChannels,
        [FromQuery] string? profile,
        [FromQuery] [RegularExpression(EncodingHelper.LevelValidationRegex)] string? level,
        [FromQuery] float? framerate,
        [FromQuery] float? maxFramerate,
        [FromQuery] bool? copyTimestamps,
        [FromQuery] long? startTimeTicks,
        [FromQuery] int? width,
        [FromQuery] int? height,
        [FromQuery] int? videoBitRate,
        [FromQuery] int? subtitleStreamIndex,
        [FromQuery] SubtitleDeliveryMethod? subtitleMethod,
        [FromQuery] int? maxRefFrames,
        [FromQuery] int? maxVideoBitDepth,
        [FromQuery] bool? requireAvc,
        [FromQuery] bool? deInterlace,
        [FromQuery] bool? requireNonAnamorphic,
        [FromQuery] int? transcodingMaxAudioChannels,
        [FromQuery] int? cpuCoreLimit,
        [FromQuery] string? liveStreamId,
        [FromQuery] bool? enableMpegtsM2TsMode,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? videoCodec,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? subtitleCodec,
        [FromQuery] string? transcodeReasons,
        [FromQuery] int? audioStreamIndex,
        [FromQuery] int? videoStreamIndex,
        [FromQuery] EncodingContext? context,
        [FromQuery] Dictionary<string, string> streamOptions,
        [FromQuery] int? maxWidth,
        [FromQuery] int? maxHeight,
        [FromQuery] bool? enableSubtitlesInManifest,
        [FromQuery] bool enableAudioVbrEncoding = true,
        [FromQuery] bool alwaysBurnInSubtitleWhenTranscoding = false)
    {
```

**File:** Jellyfin.Api/Controllers/DynamicHlsController.cs (L404-520)
```csharp
    [HttpGet("Videos/{itemId}/master.m3u8")]
    [HttpHead("Videos/{itemId}/master.m3u8", Name = "HeadMasterHlsVideoPlaylist")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesPlaylistFile]
    public async Task<ActionResult> GetMasterHlsVideoPlaylist(
        [FromRoute, Required] Guid itemId,
        [FromQuery] bool? @static,
        [FromQuery] string? @params,
        [FromQuery] string? tag,
        [FromQuery, ParameterObsolete] string? deviceProfileId,
        [FromQuery] string? playSessionId,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? segmentContainer,
        [FromQuery] int? segmentLength,
        [FromQuery] int? minSegments,
        [FromQuery, Required] string mediaSourceId,
        [FromQuery] string? deviceId,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? audioCodec,
        [FromQuery] bool? enableAutoStreamCopy,
        [FromQuery] bool? allowVideoStreamCopy,
        [FromQuery] bool? allowAudioStreamCopy,
        [FromQuery] int? audioSampleRate,
        [FromQuery] int? maxAudioBitDepth,
        [FromQuery] int? audioBitRate,
        [FromQuery] int? audioChannels,
        [FromQuery] int? maxAudioChannels,
        [FromQuery] string? profile,
        [FromQuery] [RegularExpression(EncodingHelper.LevelValidationRegex)] string? level,
        [FromQuery] float? framerate,
        [FromQuery] float? maxFramerate,
        [FromQuery] bool? copyTimestamps,
        [FromQuery] long? startTimeTicks,
        [FromQuery] int? width,
        [FromQuery] int? height,
        [FromQuery] int? maxWidth,
        [FromQuery] int? maxHeight,
        [FromQuery] int? videoBitRate,
        [FromQuery] int? subtitleStreamIndex,
        [FromQuery] SubtitleDeliveryMethod? subtitleMethod,
        [FromQuery] int? maxRefFrames,
        [FromQuery] int? maxVideoBitDepth,
        [FromQuery] bool? requireAvc,
        [FromQuery] bool? deInterlace,
        [FromQuery] bool? requireNonAnamorphic,
        [FromQuery] int? transcodingMaxAudioChannels,
        [FromQuery] int? cpuCoreLimit,
        [FromQuery] string? liveStreamId,
        [FromQuery] bool? enableMpegtsM2TsMode,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? videoCodec,
        [FromQuery] [RegularExpression(EncodingHelper.ContainerValidationRegex)] string? subtitleCodec,
        [FromQuery] string? transcodeReasons,
        [FromQuery] int? audioStreamIndex,
        [FromQuery] int? videoStreamIndex,
        [FromQuery] EncodingContext? context,
        [FromQuery] Dictionary<string, string> streamOptions,
        [FromQuery] bool enableAdaptiveBitrateStreaming = false,
        [FromQuery] bool enableTrickplay = true,
        [FromQuery] bool enableAudioVbrEncoding = true,
        [FromQuery] bool alwaysBurnInSubtitleWhenTranscoding = false)
    {
        var streamingRequest = new HlsVideoRequestDto
        {
            Id = itemId,
            Static = @static ?? false,
            Params = @params,
            Tag = tag,
            PlaySessionId = playSessionId,
            SegmentContainer = segmentContainer,
            SegmentLength = segmentLength,
            MinSegments = minSegments,
            MediaSourceId = mediaSourceId,
            DeviceId = deviceId,
            AudioCodec = audioCodec,
            EnableAutoStreamCopy = enableAutoStreamCopy ?? true,
            AllowAudioStreamCopy = allowAudioStreamCopy ?? true,
            AllowVideoStreamCopy = allowVideoStreamCopy ?? true,
            AudioSampleRate = audioSampleRate,
            MaxAudioChannels = maxAudioChannels,
            AudioBitRate = audioBitRate,
            MaxAudioBitDepth = maxAudioBitDepth,
            AudioChannels = audioChannels,
            Profile = profile,
            Level = level,
            Framerate = framerate,
            MaxFramerate = maxFramerate,
            CopyTimestamps = copyTimestamps ?? false,
            StartTimeTicks = startTimeTicks,
            Width = width,
            Height = height,
            MaxWidth = maxWidth,
            MaxHeight = maxHeight,
            VideoBitRate = videoBitRate,
            SubtitleStreamIndex = subtitleStreamIndex,
            SubtitleMethod = subtitleMethod ?? SubtitleDeliveryMethod.External,
            MaxRefFrames = maxRefFrames,
            MaxVideoBitDepth = maxVideoBitDepth,
            RequireAvc = requireAvc ?? false,
            DeInterlace = deInterlace ?? false,
            RequireNonAnamorphic = requireNonAnamorphic ?? false,
            TranscodingMaxAudioChannels = transcodingMaxAudioChannels,
            CpuCoreLimit = cpuCoreLimit,
            LiveStreamId = liveStreamId,
            EnableMpegtsM2TsMode = enableMpegtsM2TsMode ?? false,
            VideoCodec = videoCodec,
            SubtitleCodec = subtitleCodec,
            TranscodeReasons = transcodeReasons,
            AudioStreamIndex = audioStreamIndex,
            VideoStreamIndex = videoStreamIndex,
            Context = context ?? EncodingContext.Streaming,
            StreamOptions = streamOptions,
            EnableAdaptiveBitrateStreaming = enableAdaptiveBitrateStreaming,
            EnableTrickplay = enableTrickplay,
            EnableAudioVbrEncoding = enableAudioVbrEncoding,
            AlwaysBurnInSubtitleWhenTranscoding = alwaysBurnInSubtitleWhenTranscoding
        };

        return await _dynamicHlsHelper.GetMasterHlsPlaylist(TranscodingJobType, streamingRequest, enableAdaptiveBitrateStreaming).ConfigureAwait(false);
    }
```

**File:** Jellyfin.Api/Controllers/DynamicHlsController.cs (L800-859)
```csharp
    {
        using var cancellationTokenSource = new CancellationTokenSource();
        var streamingRequest = new VideoRequestDto
        {
            Id = itemId,
            Static = @static ?? false,
            Params = @params,
            Tag = tag,
            PlaySessionId = playSessionId,
            SegmentContainer = segmentContainer,
            SegmentLength = segmentLength,
            MinSegments = minSegments,
            MediaSourceId = mediaSourceId,
            DeviceId = deviceId,
            AudioCodec = audioCodec,
            EnableAutoStreamCopy = enableAutoStreamCopy ?? true,
            AllowAudioStreamCopy = allowAudioStreamCopy ?? true,
            AllowVideoStreamCopy = allowVideoStreamCopy ?? true,
            AudioSampleRate = audioSampleRate,
            MaxAudioChannels = maxAudioChannels,
            AudioBitRate = audioBitRate,
            MaxAudioBitDepth = maxAudioBitDepth,
            AudioChannels = audioChannels,
            Profile = profile,
            Level = level,
            Framerate = framerate,
            MaxFramerate = maxFramerate,
            CopyTimestamps = copyTimestamps ?? false,
            StartTimeTicks = startTimeTicks,
            Width = width,
            Height = height,
            MaxWidth = maxWidth,
            MaxHeight = maxHeight,
            VideoBitRate = videoBitRate,
            SubtitleStreamIndex = subtitleStreamIndex,
            SubtitleMethod = subtitleMethod ?? SubtitleDeliveryMethod.External,
            MaxRefFrames = maxRefFrames,
            MaxVideoBitDepth = maxVideoBitDepth,
            RequireAvc = requireAvc ?? false,
            DeInterlace = deInterlace ?? false,
            RequireNonAnamorphic = requireNonAnamorphic ?? false,
            TranscodingMaxAudioChannels = transcodingMaxAudioChannels,
            CpuCoreLimit = cpuCoreLimit,
            LiveStreamId = liveStreamId,
            EnableMpegtsM2TsMode = enableMpegtsM2TsMode ?? false,
            VideoCodec = videoCodec,
            SubtitleCodec = subtitleCodec,
            TranscodeReasons = transcodeReasons,
            AudioStreamIndex = audioStreamIndex,
            VideoStreamIndex = videoStreamIndex,
            Context = context ?? EncodingContext.Streaming,
            StreamOptions = streamOptions,
            EnableAudioVbrEncoding = enableAudioVbrEncoding,
            AlwaysBurnInSubtitleWhenTranscoding = alwaysBurnInSubtitleWhenTranscoding
        };

        return await GetVariantPlaylistInternal(streamingRequest, cancellationTokenSource)
            .ConfigureAwait(false);
    }

```

**File:** Jellyfin.Api/Controllers/DynamicHlsController.cs (L1327-1385)
```csharp
    {
        var streamingRequest = new StreamingRequestDto
        {
            Id = itemId,
            Container = container,
            CurrentRuntimeTicks = runtimeTicks,
            ActualSegmentLengthTicks = actualSegmentLengthTicks,
            Static = @static ?? false,
            Params = @params,
            Tag = tag,
            PlaySessionId = playSessionId,
            SegmentContainer = segmentContainer,
            SegmentLength = segmentLength,
            MinSegments = minSegments,
            MediaSourceId = mediaSourceId,
            DeviceId = deviceId,
            AudioCodec = audioCodec,
            EnableAutoStreamCopy = enableAutoStreamCopy ?? true,
            AllowAudioStreamCopy = allowAudioStreamCopy ?? true,
            AllowVideoStreamCopy = allowVideoStreamCopy ?? true,
            AudioSampleRate = audioSampleRate,
            MaxAudioChannels = maxAudioChannels,
            AudioBitRate = audioBitRate ?? maxStreamingBitrate,
            MaxAudioBitDepth = maxAudioBitDepth,
            AudioChannels = audioChannels,
            Profile = profile,
            Level = level,
            Framerate = framerate,
            MaxFramerate = maxFramerate,
            CopyTimestamps = copyTimestamps ?? false,
            StartTimeTicks = startTimeTicks,
            Width = width,
            Height = height,
            VideoBitRate = videoBitRate,
            SubtitleStreamIndex = subtitleStreamIndex,
            SubtitleMethod = subtitleMethod ?? SubtitleDeliveryMethod.External,
            MaxRefFrames = maxRefFrames,
            MaxVideoBitDepth = maxVideoBitDepth,
            RequireAvc = requireAvc ?? false,
            DeInterlace = deInterlace ?? false,
            RequireNonAnamorphic = requireNonAnamorphic ?? false,
            TranscodingMaxAudioChannels = transcodingMaxAudioChannels,
            CpuCoreLimit = cpuCoreLimit,
            LiveStreamId = liveStreamId,
            EnableMpegtsM2TsMode = enableMpegtsM2TsMode ?? false,
            VideoCodec = videoCodec,
            SubtitleCodec = subtitleCodec,
            TranscodeReasons = transcodeReasons,
            AudioStreamIndex = audioStreamIndex,
            VideoStreamIndex = videoStreamIndex,
            Context = context ?? EncodingContext.Streaming,
            StreamOptions = streamOptions,
            EnableAudioVbrEncoding = enableAudioVbrEncoding,
            AlwaysBurnInSubtitleWhenTranscoding = false
        };

        return await GetDynamicSegment(streamingRequest, segmentId)
            .ConfigureAwait(false);
    }
```

**File:** Jellyfin.Api/Controllers/DynamicHlsController.cs (L1460-1547)
```csharp
        if (System.IO.File.Exists(segmentPath))
        {
            job = _transcodeManager.OnTranscodeBeginRequest(playlistPath, TranscodingJobType);
            _logger.LogDebug("returning {0} [it exists, try 1]", segmentPath);
            return await GetSegmentResult(state, playlistPath, segmentPath, segmentExtension, segmentId, job, cancellationToken).ConfigureAwait(false);
        }

        using (await _transcodeManager.LockAsync(playlistPath, cancellationToken).ConfigureAwait(false))
        {
            var startTranscoding = false;
            if (System.IO.File.Exists(segmentPath))
            {
                job = _transcodeManager.OnTranscodeBeginRequest(playlistPath, TranscodingJobType);
                _logger.LogDebug("returning {0} [it exists, try 2]", segmentPath);
                return await GetSegmentResult(state, playlistPath, segmentPath, segmentExtension, segmentId, job, cancellationToken).ConfigureAwait(false);
            }

            var currentTranscodingIndex = GetCurrentTranscodingIndex(playlistPath, segmentExtension);
            var segmentGapRequiringTranscodingChange = 24 / state.SegmentLength;

            if (segmentId == -1)
            {
                _logger.LogDebug("Starting transcoding because fmp4 init file is being requested");
                startTranscoding = true;
                segmentId = 0;
            }
            else if (currentTranscodingIndex is null)
            {
                _logger.LogDebug("Starting transcoding because currentTranscodingIndex=null");
                startTranscoding = true;
            }
            else if (segmentId < currentTranscodingIndex.Value)
            {
                _logger.LogDebug("Starting transcoding because requestedIndex={0} and currentTranscodingIndex={1}", segmentId, currentTranscodingIndex);
                startTranscoding = true;
            }
            else if (segmentId - currentTranscodingIndex.Value > segmentGapRequiringTranscodingChange)
            {
                _logger.LogDebug("Starting transcoding because segmentGap is {0} and max allowed gap is {1}. requestedIndex={2}", segmentId - currentTranscodingIndex.Value, segmentGapRequiringTranscodingChange, segmentId);
                startTranscoding = true;
            }

            if (startTranscoding)
            {
                // If the playlist doesn't already exist, startup ffmpeg
                try
                {
                    await _transcodeManager.KillTranscodingJobs(streamingRequest.DeviceId, streamingRequest.PlaySessionId, p => false)
                        .ConfigureAwait(false);

                    if (currentTranscodingIndex.HasValue)
                    {
                        await DeleteLastFile(playlistPath, segmentExtension, 0).ConfigureAwait(false);
                    }

                    streamingRequest.StartTimeTicks = streamingRequest.CurrentRuntimeTicks;

                    state.WaitForPath = segmentPath;
                    job = await _transcodeManager.StartFfMpeg(
                        state,
                        playlistPath,
                        GetCommandLineArguments(playlistPath, state, false, segmentId),
                        Request.HttpContext.User.GetUserId(),
                        TranscodingJobType,
                        cancellationTokenSource).ConfigureAwait(false);
                }
                catch
                {
                    state.Dispose();
                    throw;
                }

                // await WaitForMinimumSegmentCount(playlistPath, 1, cancellationTokenSource.Token).ConfigureAwait(false);
            }
            else
            {
                job = _transcodeManager.OnTranscodeBeginRequest(playlistPath, TranscodingJobType);
                if (job?.TranscodingThrottler is not null)
                {
                    await job.TranscodingThrottler.UnpauseTranscoding().ConfigureAwait(false);
                }
            }
        }

        _logger.LogDebug("returning {0} [general case]", segmentPath);
        job ??= _transcodeManager.OnTranscodeBeginRequest(playlistPath, TranscodingJobType);
        return await GetSegmentResult(state, playlistPath, segmentPath, segmentExtension, segmentId, job, cancellationToken).ConfigureAwait(false);
    }
```

**File:** Jellyfin.Api/Controllers/VideosController.cs (L424-490)
```csharp
        var state = await StreamingHelpers.GetStreamingState(
                streamingRequest,
                HttpContext,
                _mediaSourceManager,
                _userManager,
                _libraryManager,
                _serverConfigurationManager,
                _mediaEncoder,
                _encodingHelper,
                _transcodeManager,
                _transcodingJobType,
                cancellationTokenSource.Token)
            .ConfigureAwait(false);

        if (@static.HasValue && @static.Value && state.DirectStreamProvider is not null)
        {
            var liveStreamInfo = _mediaSourceManager.GetLiveStreamInfo(streamingRequest.LiveStreamId);
            if (liveStreamInfo is null)
            {
                return NotFound();
            }

            var liveStream = new ProgressiveFileStream(liveStreamInfo.GetStream());
            // TODO (moved from MediaBrowser.Api): Don't hardcode contentType
            return File(liveStream, MimeTypes.GetMimeType("file.ts"));
        }

        // Static remote stream
        if (@static.HasValue && @static.Value && state.InputProtocol == MediaProtocol.Http)
        {
            var httpClient = _httpClientFactory.CreateClient(NamedClient.Default);
            return await FileStreamResponseHelpers.GetStaticRemoteStreamResult(state, httpClient, HttpContext).ConfigureAwait(false);
        }

        if (@static.HasValue && @static.Value && state.InputProtocol != MediaProtocol.File)
        {
            return BadRequest($"Input protocol {state.InputProtocol} cannot be streamed statically");
        }

        // Static stream
        if (@static.HasValue && @static.Value && !(state.MediaSource.VideoType == VideoType.BluRay || state.MediaSource.VideoType == VideoType.Dvd))
        {
            var contentType = state.GetMimeType("." + state.OutputContainer, false) ?? state.GetMimeType(state.MediaPath);

            if (state.MediaSource.IsInfiniteStream)
            {
                var liveStream = new ProgressiveFileStream(state.MediaPath, null, _transcodeManager);
                return File(liveStream, contentType);
            }

            return FileStreamResponseHelpers.GetStaticFileResult(
                state.MediaPath,
                contentType);
        }

        // Need to start ffmpeg (because media can't be returned directly)
        var encodingOptions = _serverConfigurationManager.GetEncodingOptions();
        var ffmpegCommandLineArguments = _encodingHelper.GetProgressiveVideoFullCommandLine(state, encodingOptions, EncoderPreset.superfast);
        return await FileStreamResponseHelpers.GetTranscodedFile(
            state,
            isHeadRequest,
            HttpContext,
            _transcodeManager,
            ffmpegCommandLineArguments,
            _transcodingJobType,
            cancellationTokenSource).ConfigureAwait(false);
    }
```

**File:** Jellyfin.Api/Controllers/UniversalAudioController.cs (L165-229)
```csharp
        _mediaInfoHelper.SortMediaSources(info, maxStreamingBitrate);

        foreach (var source in info.MediaSources)
        {
            _mediaInfoHelper.NormalizeMediaSourceContainer(source, deviceProfile, DlnaProfileType.Video);
        }

        var mediaSource = info.MediaSources[0];
        if (mediaSource.SupportsDirectPlay && mediaSource.Protocol == MediaProtocol.Http && enableRedirection && mediaSource.IsRemote && enableRemoteMedia.HasValue && enableRemoteMedia.Value)
        {
            return Redirect(mediaSource.Path);
        }

        // This one is currently very misleading as the SupportsDirectStream actually means "can direct play"
        // The definition of DirectStream also seems changed during development
        var isStatic = mediaSource.SupportsDirectStream;
        if (!isStatic && mediaSource.TranscodingSubProtocol == MediaStreamProtocol.hls)
        {
            // hls segment container can only be mpegts or fmp4 per ffmpeg documentation
            // ffmpeg option -> file extension
            //        mpegts -> ts
            //          fmp4 -> mp4
            var supportedHlsContainers = new[] { "ts", "mp4" };

            // fallback to mpegts if device reports some weird value unsupported by hls
            var requestedSegmentContainer = Array.Exists(
                supportedHlsContainers,
                element => string.Equals(element, transcodingContainer, StringComparison.OrdinalIgnoreCase)) ? transcodingContainer : "ts";
            var segmentContainer = Array.Exists(
                supportedHlsContainers,
                element => string.Equals(element, mediaSource.TranscodingContainer, StringComparison.OrdinalIgnoreCase)) ? mediaSource.TranscodingContainer : requestedSegmentContainer;
            var dynamicHlsRequestDto = new HlsAudioRequestDto
            {
                Id = itemId,
                Container = ".m3u8",
                Static = isStatic,
                PlaySessionId = info.PlaySessionId,
                SegmentContainer = segmentContainer,
                MediaSourceId = mediaSourceId,
                DeviceId = deviceId,
                AudioCodec = mediaSource.TranscodeReasons == TranscodeReason.ContainerNotSupported ? "copy" : audioCodec,
                EnableAutoStreamCopy = true,
                AllowAudioStreamCopy = true,
                AllowVideoStreamCopy = true,
                AudioSampleRate = maxAudioSampleRate,
                MaxAudioChannels = maxAudioChannels,
                MaxAudioBitDepth = maxAudioBitDepth,
                AudioBitRate = audioBitRate ?? maxStreamingBitrate,
                StartTimeTicks = startTimeTicks,
                SubtitleMethod = SubtitleDeliveryMethod.Hls,
                RequireAvc = false,
                DeInterlace = false,
                RequireNonAnamorphic = false,
                EnableMpegtsM2TsMode = false,
                TranscodeReasons = mediaSource.TranscodeReasons == 0 ? null : mediaSource.TranscodeReasons.ToString(),
                Context = EncodingContext.Static,
                StreamOptions = new Dictionary<string, string>(),
                EnableAdaptiveBitrateStreaming = false,
                EnableAudioVbrEncoding = enableAudioVbrEncoding
            };

            return await _dynamicHlsHelper.GetMasterHlsPlaylist(TranscodingJobType.Hls, dynamicHlsRequestDto, true)
                .ConfigureAwait(false);
        }

```

**File:** MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs (L448-478)
```csharp
        _logger.LogInformation("{Filename} {Arguments}", process.StartInfo.FileName, process.StartInfo.Arguments);

        var logFilePrefix = "FFmpeg.Transcode-";
        if (state.VideoRequest is not null
            && EncodingHelper.IsCopyCodec(state.OutputVideoCodec))
        {
            logFilePrefix = EncodingHelper.IsCopyCodec(state.OutputAudioCodec)
                ? "FFmpeg.Remux-"
                : "FFmpeg.DirectStream-";
        }

        if (state.VideoRequest is null && EncodingHelper.IsCopyCodec(state.OutputAudioCodec))
        {
            logFilePrefix = "FFmpeg.Remux-";
        }

        var logFilePath = Path.Combine(
            _serverConfigurationManager.ApplicationPaths.LogDirectoryPath,
            $"{logFilePrefix}{DateTime.Now:yyyy-MM-dd_HH-mm-ss}_{state.Request.MediaSourceId}_{Guid.NewGuid().ToString()[..8]}.log");

        // FFmpeg writes debug/error info to stderr. This is useful when debugging so let's put it in the log directory.
        Stream logStream = new FileStream(
            logFilePath,
            FileMode.Create,
            FileAccess.Write,
            FileShare.Read,
            IODefaults.FileStreamBufferSize,
            FileOptions.Asynchronous);

        await JsonSerializer.SerializeAsync(logStream, state.MediaSource, cancellationToken: cancellationTokenSource.Token).ConfigureAwait(false);
        var commandLineLogMessageBytes = Encoding.UTF8.GetBytes(
```

**File:** MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs (L542-551)
```csharp
    private void StartThrottler(StreamState state, TranscodingJob transcodingJob)
    {
        if (EnableThrottling(state)
            && (_mediaEncoder.IsPkeyPauseSupported
                || _mediaEncoder.EncoderVersion <= _maxFFmpegCkeyPauseSupported))
        {
            transcodingJob.TranscodingThrottler = new TranscodingThrottler(transcodingJob, _loggerFactory.CreateLogger<TranscodingThrottler>(), _serverConfigurationManager, _fileSystem, _mediaEncoder);
            transcodingJob.TranscodingThrottler.Start();
        }
    }
```

**File:** MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs (L560-567)
```csharp
    private void StartSegmentCleaner(StreamState state, TranscodingJob transcodingJob)
    {
        if (EnableSegmentCleaning(state))
        {
            transcodingJob.TranscodingSegmentCleaner = new TranscodingSegmentCleaner(transcodingJob, _loggerFactory.CreateLogger<TranscodingSegmentCleaner>(), _serverConfigurationManager, _fileSystem, _mediaEncoder, state.SegmentLength);
            transcodingJob.TranscodingSegmentCleaner.Start();
        }
    }
```

**File:** MediaBrowser.MediaEncoding/Transcoding/TranscodeManager.cs (L576-609)
```csharp
    private TranscodingJob OnTranscodeBeginning(
        string path,
        string? playSessionId,
        string? liveStreamId,
        string transcodingJobId,
        TranscodingJobType type,
        Process process,
        string? deviceId,
        StreamState state,
        CancellationTokenSource cancellationTokenSource)
    {
        lock (_activeTranscodingJobs)
        {
            var job = new TranscodingJob(_loggerFactory.CreateLogger<TranscodingJob>())
            {
                Type = type,
                Path = path,
                Process = process,
                ActiveRequestCount = 1,
                DeviceId = deviceId,
                CancellationTokenSource = cancellationTokenSource,
                Id = transcodingJobId,
                PlaySessionId = playSessionId,
                LiveStreamId = liveStreamId,
                MediaSource = state.MediaSource
            };

            _activeTranscodingJobs.Add(job);

            ReportTranscodingProgress(job, state, null, null, null, null, null);

            return job;
        }
    }
```

**File:** Jellyfin.Api/Helpers/DynamicHlsHelper.cs (L96-177)
```csharp
    public async Task<ActionResult> GetMasterHlsPlaylist(
        TranscodingJobType transcodingJobType,
        StreamingRequestDto streamingRequest,
        bool enableAdaptiveBitrateStreaming)
    {
        var isHeadRequest = _httpContextAccessor.HttpContext?.Request.Method == WebRequestMethods.Http.Head;
        // CTS lifecycle is managed internally.
        var cancellationTokenSource = new CancellationTokenSource();
        return await GetMasterPlaylistInternal(
            streamingRequest,
            isHeadRequest,
            enableAdaptiveBitrateStreaming,
            transcodingJobType,
            cancellationTokenSource).ConfigureAwait(false);
    }

    private async Task<ActionResult> GetMasterPlaylistInternal(
        StreamingRequestDto streamingRequest,
        bool isHeadRequest,
        bool enableAdaptiveBitrateStreaming,
        TranscodingJobType transcodingJobType,
        CancellationTokenSource cancellationTokenSource)
    {
        if (_httpContextAccessor.HttpContext is null)
        {
            throw new ResourceNotFoundException(nameof(_httpContextAccessor.HttpContext));
        }

        using var state = await StreamingHelpers.GetStreamingState(
                streamingRequest,
                _httpContextAccessor.HttpContext,
                _mediaSourceManager,
                _userManager,
                _libraryManager,
                _serverConfigurationManager,
                _mediaEncoder,
                _encodingHelper,
                _transcodeManager,
                transcodingJobType,
                cancellationTokenSource.Token)
            .ConfigureAwait(false);

        _httpContextAccessor.HttpContext.Response.Headers.Append(HeaderNames.Expires, "0");
        if (isHeadRequest)
        {
            return new FileContentResult(Array.Empty<byte>(), MimeTypes.GetMimeType("playlist.m3u8"));
        }

        var totalBitrate = (state.OutputAudioBitrate ?? 0) + (state.OutputVideoBitrate ?? 0);

        var builder = new StringBuilder();

        builder.AppendLine("#EXTM3U");

        var isLiveStream = state.IsSegmentedLiveStream;

        var queryString = _httpContextAccessor.HttpContext.Request.QueryString.ToString();

        // from universal audio service, need to override the AudioCodec when the actual request differs from original query
        if (!string.Equals(state.OutputAudioCodec, _httpContextAccessor.HttpContext.Request.Query["AudioCodec"].ToString(), StringComparison.OrdinalIgnoreCase))
        {
            var newQuery = Microsoft.AspNetCore.WebUtilities.QueryHelpers.ParseQuery(queryString);
            newQuery["AudioCodec"] = state.OutputAudioCodec;
            queryString = Microsoft.AspNetCore.WebUtilities.QueryHelpers.AddQueryString(string.Empty, newQuery);
        }

        // from universal audio service
        if (!string.IsNullOrWhiteSpace(state.Request.SegmentContainer)
            && !queryString.Contains("SegmentContainer", StringComparison.OrdinalIgnoreCase))
        {
            queryString += "&SegmentContainer=" + state.Request.SegmentContainer;
        }

        // from universal audio service
        if (!string.IsNullOrWhiteSpace(state.Request.TranscodeReasons)
            && !queryString.Contains("TranscodeReasons=", StringComparison.OrdinalIgnoreCase))
        {
            queryString += "&TranscodeReasons=" + state.Request.TranscodeReasons;
        }

        // Video rotation metadata is only supported in fMP4 remuxing
        if (state.VideoStream is not null
```


## Jellyfin播放链路实现
本代码图展示了Jellyfin媒体服务器的播放链路实现，包括HLS流、渐进式流和转码机制。关键组件包括：[1a]DynamicHlsController处理HLS请求、[2a]StreamingHelpers构建流状态、[3a]TranscodeManager管理FFmpeg进程、[5a]VideosController处理渐进式流、[6a]转码控制优化机制。
### 1. HLS播放请求处理流程
展示客户端请求HLS播放时，从Controller到FFmpeg启动的完整流程
### 1a. 接收主播放列表请求 (`DynamicHlsController.cs:408`)
客户端请求master.m3u8主播放列表
```text
public async Task<ActionResult> GetMasterHlsVideoPlaylist(
```
### 1b. 委托给Helper处理 (`DynamicHlsController.cs:519`)
调用DynamicHlsHelper生成播放列表
```text
return await _dynamicHlsHelper.GetMasterHlsPlaylist(TranscodingJobType, streamingRequest, enableAdaptiveBitrateStreaming).ConfigureAwait(false);
```
### 1c. 构建流状态 (`DynamicHlsHelper.cs:124`)
获取媒体源信息和编码参数
```text
using var state = await StreamingHelpers.GetStreamingState(
```
### 1d. 启动FFmpeg转码 (`DynamicHlsController.cs:309`)
启动FFmpeg进程进行HLS转码
```text
job = await _transcodeManager.StartFfMpeg(
```
### 2. 流状态构建与媒体源选择
展示如何根据请求参数和设备能力选择合适的播放模式和媒体源
### 2a. 获取流状态入口 (`StreamingHelpers.cs:47`)
所有播放请求都会调用的核心方法
```text
public static async Task<StreamState> GetStreamingState(
```
### 2b. 获取媒体源列表 (`StreamingHelpers.cs:131`)
从媒体库获取可用的播放源
```text
var mediaSources = await mediaSourceManager.GetPlaybackMediaSources(
```
### 2c. 附加媒体源信息 (`StreamingHelpers.cs:158`)
将媒体源信息附加到流状态
```text
encodingHelper.AttachMediaSourceInfo(state, encodingOptions, mediaSource, url);
```
### 2d. 尝试直接流复制 (`StreamingHelpers.cs:204`)
判断是否可以直接复制流而不重新编码
```text
encodingHelper.TryStreamCopy(state, encodingOptions);
```
### 3. FFmpeg转码进程管理
展示TranscodeManager如何启动和管理FFmpeg进程
### 3a. 启动FFmpeg入口 (`TranscodeManager.cs:371`)
启动FFmpeg转码进程的核心方法
```text
public async Task<TranscodingJob> StartFfMpeg(
```
### 3b. 创建FFmpeg进程 (`TranscodeManager.cs:417`)
配置并创建FFmpeg进程对象
```text
var process = new Process
```
### 3c. 设置编码器路径 (`TranscodeManager.cs:429`)
指定FFmpeg可执行文件路径
```text
FileName = _mediaEncoder.EncoderPath,
```
### 3d. 设置命令行参数 (`TranscodeManager.cs:430`)
传入构建好的FFmpeg命令行参数
```text
Arguments = commandLineArguments,
```
### 3e. 启动进程 (`TranscodeManager.cs:491`)
正式启动FFmpeg进程
```text
process.Start();
```
### 4. HLS分段请求处理
展示客户端请求HLS分段时的处理流程
### 4a. 接收分段请求 (`DynamicHlsController.cs:1327`)
客户端请求具体的ts/mp4分段文件
```text
public async Task<ActionResult> GetDynamicSegment(
```
### 4b. 检查分段存在 (`DynamicHlsController.cs:1460`)
检查分段文件是否已存在
```text
if (System.IO.File.Exists(segmentPath))
```
### 4c. 触发转码启动 (`DynamicHlsController.cs:1514`)
分段不存在时启动转码
```text
startTranscoding = true;
```
### 4d. 返回分段结果 (`DynamicHlsController.cs:1546`)
向客户端返回分段文件
```text
return await GetSegmentResult(state, playlistPath, segmentPath, segmentExtension, segmentId, job, cancellationToken).ConfigureAwait(false);
```
### 5. 渐进式视频流处理
展示VideosController如何处理渐进式视频流请求
### 5a. 接收渐进式流请求 (`VideosController.cs:314`)
客户端请求/video/{id}/stream
```text
public async Task<ActionResult> GetVideoStream(
```
### 5b. 构建流状态 (`VideosController.cs:424`)
获取媒体信息和编码参数
```text
var state = await StreamingHelpers.GetStreamingState(
```
### 5c. 静态文件返回 (`VideosController.cs:474`)
直接返回原始文件（DirectPlay）
```text
return FileStreamResponseHelpers.GetStaticFileResult(
```
### 5d. 转码文件返回 (`VideosController.cs:482`)
启动转码并返回转码后的文件
```text
return await FileStreamResponseHelpers.GetTranscodedFile(
```
### 6. 转码控制与优化
展示转码过程中的节流控制和分段清理机制
### 6a. 启动节流器 (`TranscodeManager.cs:529`)
根据播放状态控制转码速度
```text
StartThrottler(state, transcodingJob);
```
### 6b. 启动分段清理器 (`TranscodeManager.cs:530`)
定期清理过期的HLS分段文件
```text
StartSegmentCleaner(state, transcodingJob);
```
### 6c. 节流器定时器 (`TranscodingThrottler.cs:46`)
每5秒检查一次播放状态
```text
_timer = new Timer(TimerCallback, null, 5000, 5000);
```
### 6d. 发送恢复命令 (`TranscodingThrottler.cs:62`)
向FFmpeg发送恢复转码命令
```text
await _job.Process!.StandardInput.WriteAsync(resumeKey).ConfigureAwait(false);
```