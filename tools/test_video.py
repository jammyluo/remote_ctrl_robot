import cv2
import sys
import platform

def check_system_compatibility():
    """检查系统兼容性"""
    system = platform.system()
    print(f"当前系统: {system}")
    
    if system == "Darwin":  # macOS
        print("检测到苹果系统，可能需要特殊配置")
        return "macos"
    elif system == "Linux":
        print("Linux系统，支持GStreamer")
        return "linux"
    else:
        print("未知系统")
        return "unknown"

def test_opencv_gstreamer():
    """测试OpenCV是否支持GStreamer"""
    try:
        # 测试GStreamer后端是否可用
        test_cap = cv2.VideoCapture("videotestsrc ! appsink", cv2.CAP_GSTREAMER)
        if test_cap.isOpened():
            test_cap.release()
            print("OpenCV GStreamer支持正常")
            return True
        else:
            print("OpenCV GStreamer支持不可用")
            return False
    except Exception as e:
        print(f"GStreamer测试失败: {e}")
        return False

def main():
    system_type = check_system_compatibility()
    
    if not test_opencv_gstreamer():
        print("错误: OpenCV不支持GStreamer后端")
        print("解决方案:")
        print("1. 安装GStreamer: brew install gstreamer gst-plugins-base gst-plugins-good gst-plugins-bad gst-plugins-ugly")
        print("2. 重新编译OpenCV with GStreamer支持")
        print("3. 使用替代方案（如FFmpeg后端）")
        return
    
    # 苹果系统特殊配置
    if system_type == "macos":
        # 设置窗口属性
        cv2.namedWindow("Input via Gstreamer", cv2.WINDOW_NORMAL)
        cv2.resizeWindow("Input via Gstreamer", 1280, 720)
    
    # 修改后的GStreamer管道，适配苹果系统
    # 注意：需要替换<interface_name>为实际的网络接口名称
    gstreamer_str = (
        "udpsrc address=230.1.1.1 port=1720 multicast-iface=lo0 ! "
        "application/x-rtp, media=video, encoding-name=H264 ! "
        "rtph264depay ! h264parse ! avdec_h264 ! "
        "videoconvert ! video/x-raw,width=1280,height=720,format=BGR ! "
        "appsink drop=1"
    )
    
    print(f"使用GStreamer管道: {gstreamer_str}")
    
    try:
        cap = cv2.VideoCapture(gstreamer_str, cv2.CAP_GSTREAMER)
        
        if not cap.isOpened():
            print("错误: 无法打开视频流")
            print("可能的原因:")
            print("1. 网络接口名称不正确（当前使用lo0）")
            print("2. 多播地址不可达")
            print("3. 端口被占用或防火墙阻止")
            return
        
        print("视频流打开成功，开始显示...")
        
        while cap.isOpened():
            ret, frame = cap.read()
            if ret:
                cv2.imshow("Input via Gstreamer", frame)
                if cv2.waitKey(25) & 0xFF == ord('q'):
                    break
            else:
                print("无法读取帧")
                break
                
    except Exception as e:
        print(f"运行时错误: {e}")
    finally:
        if 'cap' in locals():
            cap.release()
        cv2.destroyAllWindows()

if __name__ == "__main__":
    main()

# 以下是GStreamer命令行示例（已修复语法错误）

# 接收端命令（修复后的版本）:
# gst-launch-1.0 -v udpsrc port=5600 caps='application/x-rtp, media=(string)video, clock-rate=(int)90000, encoding-name=(string)H264' ! rtph264depay ! avdec_h264 ! autovideosink fps-update-interval=1000 sync=false

# 多播接收端命令:
# gst-launch-1.0 udpsrc address=230.1.1.1 port=1720 multicast-iface=lo0 ! queue ! application/x-rtp, media=video, encoding-name=H264 ! rtph264depay ! h264parse ! avdec_h264 ! videoconvert ! autovideosink

# 发送端命令:
# gst-launch-1.0 -v v4l2src device=/dev/video0 ! 'video/x-raw,width=1280,height=720,framerate=10/1' ! videoconvert ! omxh264enc ! 'video/x-h264, profile=(string)high' ! rtph264pay ! udpsink host=192.168.8.100 port=5600


# 查找当前网络接口
# networksetup -listallhardwareports | grep -A 1 "Hardware Port"

# # 加入组播组 (假设接口为 en0)
# sudo route -nv add -net 230.1.1.1 -interface en0

# # 测试组播流是否可达
# ping -c 3 230.1.1.1

# # 检查端口是否开放
# nc -zvu 230.1.1.1 1720

# # 允许 UDP 1720 端口
# sudo /usr/libexec/ApplicationFirewall/socketfilterfw \
#   --add udp:1720 \
#   --allow udp

# # 安装 FFmpeg
# brew install ffmpeg

# # 接收并播放流
# ffplay -f mpegts udp://@230.1.1.1:1720?reuse=1