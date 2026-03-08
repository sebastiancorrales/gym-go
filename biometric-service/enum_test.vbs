On Error Resume Next
' Test known enum names
WScript.Echo "DataPurposeEnrollment = " & DataPurposeEnrollment
WScript.Echo "DataPurposeVerification = " & DataPurposeVerification
WScript.Echo "CaptureFeedbackGood = " & CaptureFeedbackGood
WScript.Echo "TemplateStatusTemplateReady = " & TemplateStatusTemplateReady
WScript.Echo "TemplateStatusFailed = " & TemplateStatusFailed

' Create enrollment to see FeaturesNeeded
Set e = CreateObject("DPFPEngX.DPFPEnrollment")
If Err.Number = 0 Then
    WScript.Echo "Enrollment created OK"
    WScript.Echo "FeaturesNeeded = " & e.FeaturesNeeded
    WScript.Echo "TemplateStatus = " & e.TemplateStatus
Else
    WScript.Echo "Error creating enrollment: " & Err.Description
End If

' Test FeatureExtraction
Set fe = CreateObject("DPFPEngX.DPFPFeatureExtraction")
If Err.Number = 0 Then WScript.Echo "FeatureExtraction created OK"

' Test Verification
Set v = CreateObject("DPFPEngX.DPFPVerification")
If Err.Number = 0 Then
    WScript.Echo "Verification created OK"
    WScript.Echo "FARRequested = " & v.FARRequested
End If

' Test Template
Set t = CreateObject("DPFPShrX.DPFPTemplate")
If Err.Number = 0 Then WScript.Echo "Template created OK"
