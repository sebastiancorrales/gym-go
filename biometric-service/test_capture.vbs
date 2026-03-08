' Test VBScript - Captura de huella via COM
' Usa WScript.CreateObject con prefix de eventos (como VB6 WithEvents)

Dim done
done = False

Set Capture = WScript.CreateObject("DPFPDevX.DPFPCapture", "Capture_")
WScript.Echo "DPFPCapture creado OK"
WScript.Echo "StartCapture..."
Capture.StartCapture
WScript.Echo "Esperando dedo (30 seg)..."

Dim startTime
startTime = Timer

Do While Not done
    WScript.Sleep 100
    If Timer - startTime > 30 Then
        WScript.Echo "TIMEOUT"
        Exit Do
    End If
Loop

Capture.StopCapture
WScript.Echo "FIN"

' === Event Handlers ===
Sub Capture_OnReaderConnect(SerNum)
    WScript.Echo "READER_CONNECT: " & SerNum
End Sub

Sub Capture_OnReaderDisconnect(SerNum)
    WScript.Echo "READER_DISCONNECT: " & SerNum
End Sub

Sub Capture_OnFingerTouch(SerNum)
    WScript.Echo "FINGER_TOUCH: " & SerNum
End Sub

Sub Capture_OnFingerGone(SerNum)
    WScript.Echo "FINGER_GONE: " & SerNum
End Sub

Sub Capture_OnComplete(SerNum, Sample)
    WScript.Echo "CAPTURE_COMPLETE: " & SerNum
    done = True
End Sub

Sub Capture_OnSampleQuality(SerNum, Feedback)
    WScript.Echo "SAMPLE_QUALITY: " & Feedback
End Sub
