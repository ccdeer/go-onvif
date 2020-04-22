package onvif

import (
	"fmt"
	"net/url"
)

const mediaNameSpace = "http://www.onvif.org/ver10/media/wsdl"

var mediaXMLNs = []string{
	`xmlns:trt="http://www.onvif.org/ver10/media/wsdl"`,
	`xmlns:tt="http://www.onvif.org/ver10/schema"`,
}

// GetProfiles fetch available media profiles of ONVIF camera
func (device *Device) GetProfiles() ([]MediaProfile, error) {
	// Create SOAP
	soap := SOAP{
		Body:     "<trt:GetProfiles/>",
		XMLNs:    mediaXMLNs,
		User:     device.User,
		Password: device.Password,
	}

	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return nil, err
	}

	// Send SOAP request
	response, err := soap.SendRequest(fmt.Sprintf("http://%s/onvif/media_service", urlXAddr.Host))
	if err != nil {
		return []MediaProfile{}, err
	}

	// Get and parse list of profile to interface
	ifaceProfiles, err := response.ValuesForPath("Envelope.Body.GetProfilesResponse.Profiles")
	if err != nil {
		return []MediaProfile{}, err
	}

	// Create initial result
	result := []MediaProfile{}

	// Parse each available profile
	for _, ifaceProfile := range ifaceProfiles {
		if mapProfile, ok := ifaceProfile.(map[string]interface{}); ok {
			// Parse name and token
			profile := MediaProfile{}
			profile.Name = interfaceToString(mapProfile["Name"])
			profile.Token = interfaceToString(mapProfile["-token"])
			// Parse video source configuration
			videoSource := MediaSourceConfig{}
			if mapVideoSource, ok := mapProfile["VideoSourceConfiguration"].(map[string]interface{}); ok {
				videoSource.Name = interfaceToString(mapVideoSource["Name"])
				videoSource.Token = interfaceToString(mapVideoSource["-token"])
				videoSource.SourceToken = interfaceToString(mapVideoSource["SourceToken"])

				// Parse video bounds
				bounds := MediaBounds{}
				if mapVideoBounds, ok := mapVideoSource["Bounds"].(map[string]interface{}); ok {
					bounds.Height = interfaceToInt(mapVideoBounds["-height"])
					bounds.Width = interfaceToInt(mapVideoBounds["-width"])
				}
				videoSource.Bounds = bounds
			}
			profile.VideoSourceConfig = videoSource

			// Parse video encoder configuration
			videoEncoder := VideoEncoderConfig{}
			if mapVideoEncoder, ok := mapProfile["VideoEncoderConfiguration"].(map[string]interface{}); ok {
				videoEncoder.Name = interfaceToString(mapVideoEncoder["Name"])
				videoEncoder.Token = interfaceToString(mapVideoEncoder["-token"])
				videoEncoder.Encoding = interfaceToString(mapVideoEncoder["Encoding"])
				videoEncoder.Quality = interfaceToInt(mapVideoEncoder["Quality"])
				videoEncoder.SessionTimeout = interfaceToString(mapVideoEncoder["SessionTimeout"])

				// Parse video rate control
				rateControl := VideoRateControl{}
				if mapVideoRate, ok := mapVideoEncoder["RateControl"].(map[string]interface{}); ok {
					rateControl.BitrateLimit = interfaceToInt(mapVideoRate["BitrateLimit"])
					rateControl.EncodingInterval = interfaceToInt(mapVideoRate["EncodingInterval"])
					rateControl.FrameRateLimit = interfaceToInt(mapVideoRate["FrameRateLimit"])
				}
				videoEncoder.RateControl = rateControl

				// Parse video resolution
				resolution := MediaBounds{}
				if mapVideoRes, ok := mapVideoEncoder["Resolution"].(map[string]interface{}); ok {
					resolution.Height = interfaceToInt(mapVideoRes["Height"])
					resolution.Width = interfaceToInt(mapVideoRes["Width"])
				}
				videoEncoder.Resolution = resolution
			}
			profile.VideoEncoderConfig = videoEncoder

			// Parse audio source configuration
			audioSource := MediaSourceConfig{}
			if mapAudioSource, ok := mapProfile["AudioSourceConfiguration"].(map[string]interface{}); ok {
				audioSource.Name = interfaceToString(mapAudioSource["Name"])
				audioSource.Token = interfaceToString(mapAudioSource["-token"])
				audioSource.SourceToken = interfaceToString(mapAudioSource["SourceToken"])
			}
			profile.AudioSourceConfig = audioSource

			// Parse audio encoder configuration
			audioEncoder := AudioEncoderConfig{}
			if mapAudioEncoder, ok := mapProfile["AudioEncoderConfiguration"].(map[string]interface{}); ok {
				audioEncoder.Name = interfaceToString(mapAudioEncoder["Name"])
				audioEncoder.Token = interfaceToString(mapAudioEncoder["-token"])
				audioEncoder.Encoding = interfaceToString(mapAudioEncoder["Encoding"])
				audioEncoder.Bitrate = interfaceToInt(mapAudioEncoder["Bitrate"])
				audioEncoder.SampleRate = interfaceToInt(mapAudioEncoder["SampleRate"])
				audioEncoder.SessionTimeout = interfaceToString(mapAudioEncoder["SessionTimeout"])
			}
			profile.AudioEncoderConfig = audioEncoder

			// Parse PTZ configuration
			ptzConfig := PTZConfig{}
			if mapPTZ, ok := mapProfile["PTZConfiguration"].(map[string]interface{}); ok {
				ptzConfig.Name = interfaceToString(mapPTZ["Name"])
				ptzConfig.Token = interfaceToString(mapPTZ["-token"])
				ptzConfig.NodeToken = interfaceToString(mapPTZ["NodeToken"])
			}
			profile.PTZConfig = ptzConfig

			// Push profile to result
			result = append(result, profile)
		}
	}

	return result, nil
}

// GetStreamURI fetch stream URI of a media profile.
// Possible protocol is UDP, HTTP or RTSP
func (device *Device) GetStreamURI(profileToken, protocol string) (MediaURI, error) {
	// Create SOAP
	soap := SOAP{
		XMLNs: mediaXMLNs,
		Body: `<trt:GetStreamUri>
			<trt:StreamSetup>
				<tt:Stream>RTP-Unicast</tt:Stream>
				<tt:Transport><tt:Protocol>` + protocol + `</tt:Protocol></tt:Transport>
			</trt:StreamSetup>
			<trt:ProfileToken>` + profileToken + `</trt:ProfileToken>
		</trt:GetStreamUri>`,
		User:     device.User,
		Password: device.Password,
	}
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return MediaURI{}, err
	}

	// Send SOAP request
	response, err := soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media", urlXAddr.Host))
	if err != nil {
		return MediaURI{}, err
	}

	// Parse response to interface
	ifaceURI, err := response.ValueForPath("Envelope.Body.GetStreamUriResponse.MediaUri")
	if err != nil {
		return MediaURI{}, err
	}

	// Parse interface to struct
	streamURI := MediaURI{}
	if mapURI, ok := ifaceURI.(map[string]interface{}); ok {
		streamURI.URI = interfaceToString(mapURI["Uri"])
		streamURI.Timeout = interfaceToString(mapURI["Timeout"])
		streamURI.InvalidAfterConnect = interfaceToBool(mapURI["InvalidAfterConnect"])
		streamURI.InvalidAfterReboot = interfaceToBool(mapURI["InvalidAfterReboot"])
	}

	return streamURI, nil
}

// GetStreamURI fetch stream URI of a media profile.
// Possible protocol is UDP, HTTP or RTSP
func (device *Device) GetOSDs() ([]OSD, error) {
	// Create SOAP
	soap := SOAP{
		XMLNs:    mediaXMLNs,
		Body:     `<ns0:GetOSDs xmlns:ns0="http://www.onvif.org/ver10/media/wsdl"></ns0:GetOSDs>`,
		User:     device.User,
		Password: device.Password,
	}
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return nil, err
	}

	// Send SOAP request
	response, err := soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media", urlXAddr.Host))
	if err != nil {
		return nil, err
	}

	// Parse response to interface
	ifaceOSDs, err := response.ValuesForPath("Envelope.Body.GetOSDsResponse.OSDs")
	if err != nil {
		return nil, err
	}

	// Parse interface to struct
	result := []OSD{}

	for _, ifaceOSD := range ifaceOSDs {
		if mapOSD, ok := ifaceOSD.(map[string]interface{}); ok {
			osd := OSD{}
			osd.Token = interfaceToString(mapOSD["-token"])
			osd.VideoSourceToken = interfaceToString(mapOSD["VideoSourceConfigurationToken"])
			osd.Type = interfaceToString(mapOSD["Type"])

			pos := Position{}
			if mapPos, ok := mapOSD["Position"].(map[string]interface{}); ok {
				pos.Type = interfaceToString(mapPos["Type"])
				posXY := PosXY{}
				if mapPosXY, ok := mapPos["Pos"].(map[string]interface{}); ok {
					posXY.x = interfaceToFloat64(mapPosXY["-x"])
					posXY.y = interfaceToFloat64(mapPosXY["-y"])
				}
				pos.Pos = posXY
			}
			osd.Pos = pos

			text := TextString{}
			if mapText, ok := mapOSD["TextString"].(map[string]interface{}); ok {
				text.IsPersistentText = interfaceToBool(mapText["IsPersistentText"])
				text.Type = interfaceToString(mapText["Type"])
				text.DateFormat = interfaceToString(mapText["DateFormat"])
				text.TimeFormat = interfaceToString(mapText["TimeFormat"])
				text.FontSize = interfaceToInt(mapText["FontSize"])
				text.PlainText = interfaceToString(mapText["PlainText"])
				fontColor := OSDColor{}
				if mapFont, ok := mapText["FontColor"].(map[string]interface{}); ok {
					fontColor.Transparent = interfaceToInt(mapFont["Transparent"])
					// fontColor.Color = interfaceToString(mapFont["Color"])
				}
				text.FontColor = fontColor

				bgColor := OSDColor{}
				if mapBG, ok := mapText["FontColor"].(map[string]interface{}); ok {
					bgColor.Transparent = interfaceToInt(mapBG["Transparent"])
					// bgColor.Color = interfaceToString(mapBG["Color"])
				}
				text.BackgroundColor = bgColor

			}
			osd.Text = text
			result = append(result, osd)
		}
	}

	return result, nil
}

// <tt:FontColor><tt:Color X="0.000000" Y="0.000000" Z="0.000000" Colorspace="http://www.onvif.org/ver10/colorspace/YCbCr"/>
// </tt:FontColor>

func (device *Device) SetOSD1(token string, text string) error {
	var soap SOAP
	// Create SOAP
	soap = SOAP{
		XMLNs: mediaXMLNs,
		Body: ` <tr2:SetOSD>
				<tr2:OSD token="` + token + `"><tt:VideoSourceConfigurationToken>VideoSourceToken</tt:VideoSourceConfigurationToken>
				<tt:Type>Text</tt:Type>
				<tt:Position><tt:Type>Custom</tt:Type>
				<tt:Pos x="0.454545" y="-0.777778"/>
				</tt:Position>
				<tt:TextString><tt:Type>Plain</tt:Type>
				<tt:FontSize>16</tt:FontSize>
				<tt:PlainText>` + text + `</tt:PlainText>
				<tt:Extension><tt:ChannelName>true</tt:ChannelName>
				</tt:Extension>
			    </tt:TextString>
				</tr2:OSD>
			    </tr2:SetOSD>`,
		User:     device.User,
		Password: device.Password,
	}

	// fmt.Println(device.XAddr)
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return err
	}

	// Send SOAP request
	_, err = soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media", urlXAddr.Host))
	if err != nil {
		return err
	}
	return nil
}

func (device *Device) SetOSD(token string, text string) error {
	var soap SOAP
	// Create SOAP
	soap = SOAP{
		XMLNs:    []string{`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`, `xmlns:xsd="http://www.w3.org/2001/XMLSchema"`},
		User:     device.User,
		Password: device.Password,
	}
	soap.Body = `<SetOSD xmlns="http://www.onvif.org/ver10/media/wsdl">
		<OSD token="` + token + `">
        <VideoSourceConfigurationToken xmlns="http://www.onvif.org/ver10/schema">VideoSourceToken</VideoSourceConfigurationToken>
        <Type xmlns="http://www.onvif.org/ver10/schema">Text</Type>
        <Position xmlns="http://www.onvif.org/ver10/schema">
          <Type>Custom</Type>
		  <Pos x="0.454545" y="-0.777778" />
        </Position>
        <TextString xmlns="http://www.onvif.org/ver10/schema">
          <Type>Plain</Type>
          <FontSize>32</FontSize>
          <FontColor>
            <Color X="0.000000" Y="0.000000" Z="0.000000" Colorspace="http://www.onvif.org/ver10/colorspace/YCbCr" />
          </FontColor>
          <PlainText>` + text + `</PlainText>
	     <Extension>
	       <tt:ChannelName xmlns:tt="http://www.onvif.org/ver10/schema">true</tt:ChannelName>
	     </Extension>
        </TextString>
      </OSD>
    </SetOSD>`

	// fmt.Println(device.XAddr)
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return err
	}

	// Send SOAP request
	_, err = soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media", urlXAddr.Host))
	if err != nil {
		return err
	}
	return nil
}

func (device *Device) SetVideoEncoderConfiguration1(config VideoEncoderConfig) error {
	var soap SOAP
	// Create SOAP
	soap = SOAP{
		XMLNs:    []string{`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`, `xmlns:xsd="http://www.w3.org/2001/XMLSchema"`},
		User:     device.User,
		Password: device.Password,
	}
	soap.Body = `<SetVideoEncoderConfiguration xmlns="http://www.onvif.org/ver10/media/wsdl">
		      <Configuration token="` + config.Token + `">
		        <Name xmlns="http://www.onvif.org/ver10/schema">` + config.Name + `</Name>
		        <UseCount xmlns="http://www.onvif.org/ver10/schema">1</UseCount>
		        <GuaranteedFrameRate xmlns="http://www.onvif.org/ver10/schema">false</GuaranteedFrameRate>
		        <Encoding xmlns="http://www.onvif.org/ver10/schema">H264</Encoding>
		        <Resolution xmlns="http://www.onvif.org/ver10/schema">
		          <Width>` + fmt.Sprintf("%d", config.Resolution.Width) + `</Width>
		          <Height>` + fmt.Sprintf("%d", config.Resolution.Height) + `</Height>
		        </Resolution>
		        <H264 xmlns="http://www.onvif.org/ver10/schema">
		          <GovLength>` + fmt.Sprintf("%d", config.GovLength) + `</GovLength>
		          <H264Profile>Main</H264Profile>
		        </H264>
		        <RateControl xmlns="http://www.onvif.org/ver10/schema">
		          <FrameRateLimit>` + fmt.Sprintf("%d", config.RateControl.FrameRateLimit) + `</FrameRateLimit>
		          <EncodingInterval>` + fmt.Sprintf("%d", config.RateControl.EncodingInterval) + `</EncodingInterval>
		          <BitrateLimit>` + fmt.Sprintf("%d", config.RateControl.BitrateLimit) + `</BitrateLimit>
		        </RateControl>
		        <Multicast xmlns="http://www.onvif.org/ver10/schema">
		          <Address>
		            <Type>IPv4</Type>
		            <IPv4Address>0.0.0.0</IPv4Address>
		          </Address>
		          <Port>8860</Port>
		          <TTL>128</TTL>
		          <AutoStart>false</AutoStart>
		        </Multicast>
		        <Quality xmlns="http://www.onvif.org/ver10/schema">` + fmt.Sprintf("%d", config.Quality) + `</Quality>
				<ForcePersistence xmlns="http://www.onvif.org/ver10/schema">false</ForcePersistence>
		      </Configuration>
			</SetVideoEncoderConfiguration>`

	// fmt.Println(device.XAddr)
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return err
	}

	// Send SOAP request
	_, err = soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media", urlXAddr.Host))
	if err != nil {
		return err
	}
	return nil
}

func (device *Device) SetVideoEncoderConfiguration(config VideoEncoderConfig) error {
	var soap SOAP
	// Create SOAP
	soap = SOAP{
		XMLNs:    []string{`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`, `xmlns:xsd="http://www.w3.org/2001/XMLSchema"`},
		User:     device.User,
		Password: device.Password,
	}

	soap.Body = `<SetVideoEncoderConfiguration xmlns="http://www.onvif.org/ver20/media/wsdl">
		      <Configuration token="` + config.Token + `" GovLength="` + fmt.Sprintf("%d", config.GovLength) + `" Profile="Main">
		        <Name xmlns="http://www.onvif.org/ver10/schema">` + config.Name + `</Name>
		        <UseCount xmlns="http://www.onvif.org/ver10/schema">0</UseCount>
		        <Encoding xmlns="http://www.onvif.org/ver10/schema">H264</Encoding>
		        <Resolution xmlns="http://www.onvif.org/ver10/schema">
		          <Width>` + fmt.Sprintf("%d", config.Resolution.Width) + `</Width>
		          <Height>` + fmt.Sprintf("%d", config.Resolution.Height) + `</Height>
		        </Resolution>
		        <RateControl ConstantBitRate="false" xmlns="http://www.onvif.org/ver10/schema">
		          <FrameRateLimit>` + fmt.Sprintf("%d", config.RateControl.FrameRateLimit) + `</FrameRateLimit>
		          <BitrateLimit>` + fmt.Sprintf("%d", config.RateControl.BitrateLimit) + `</BitrateLimit>
		        </RateControl>
		        <Multicast xmlns="http://www.onvif.org/ver10/schema">
		          <Address>
		            <Type>IPv4</Type>
		            <IPv4Address>0.0.0.0</IPv4Address>
		          </Address>
		          <Port>8860</Port>
		          <TTL>128</TTL>
		          <AutoStart>false</AutoStart>
		        </Multicast>
		        <Quality xmlns="http://www.onvif.org/ver10/schema">` + fmt.Sprintf("%d", config.Quality) + `</Quality>
		      </Configuration>
			</SetVideoEncoderConfiguration>`

	// Send SOAP request
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return err
	}
	_, err = soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media2", urlXAddr.Host))
	if err != nil {
		return err
	}
	return nil
}

func (device *Device) SetAudioEncoderConfiguration(config AudioEncoderConfig) error {
    var extraObj string
	var soap SOAP
	// Create SOAP
	soap = SOAP{
		XMLNs:    []string{`xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`, `xmlns:xsd="http://www.w3.org/2001/XMLSchema"`},
		User:     device.User,
		Password: device.Password,
	}

    if config.Encoding == "AAC" {
        extraObj = `<Bitrate xmlns="http://www.onvif.org/ver10/schema">` + config.Bitrate + `</Bitrate>
                    <SampleRate xmlns="http://www.onvif.org/ver10/schema">` + config.SampleRate + `</SampleRate>`
    }

	soap.Body = `<SetAudioEncoderConfiguration xmlns="http://www.onvif.org/ver20/media/wsdl">
      <Configuration token="MainAudioEncoderToken">
        <Name xmlns="http://www.onvif.org/ver10/schema">` + config.Name + `</Name>
        <UseCount xmlns="http://www.onvif.org/ver10/schema">2</UseCount>
        <Encoding xmlns="http://www.onvif.org/ver10/schema">` + config.Encoding + `</Encoding>
        ` + extraObj + `
        <Multicast xmlns="http://www.onvif.org/ver10/schema">
          <Address>
            <Type>IPv4</Type>
            <IPv4Address>0.0.0.0</IPv4Address>
          </Address>
          <Port>8862</Port>
          <TTL>128</TTL>
          <AutoStart>false</AutoStart>
        </Multicast>
        <Bitrate xmlns="http://www.onvif.org/ver10/schema">64</Bitrate>
        <SampleRate xmlns="http://www.onvif.org/ver10/schema">8</SampleRate>
      </Configuration>
    </SetAudioEncoderConfiguration>`
	// Send SOAP request
	urlXAddr, err := url.Parse(device.XAddr)
	if err != nil {
		return err
	}
	_, err = soap.SendRequest(fmt.Sprintf("http://%s/onvif/Media2", urlXAddr.Host))
	if err != nil {
		return err
	}
	return nil
}
