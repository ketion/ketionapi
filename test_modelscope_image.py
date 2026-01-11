#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
ModelScope 图片生成测试脚本
测试 NewAPI 的 ModelScope 渠道是否正常工作
"""

import requests
import json
import time
from datetime import datetime

# ==================== 配置区域 ====================
BASE_URL = "https://api.jellal.cn"
API_KEY = "sk-tff61sXoN7AtmT6A2AjSlX0pURIqoO9SX8xwkxyu8mWzzFsX"  # 替换为你的 API Key
MODEL = "Tongyi-MAI/Z-Image-Turbo"  # 模型名称
# ==================================================


def test_image_generation():
    """测试图片生成功能"""
    
    print("=" * 60)
    print("ModelScope 图片生成测试")
    print("=" * 60)
    print(f"服务器地址: {BASE_URL}")
    print(f"模型: {MODEL}")
    print(f"测试时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print("=" * 60)
    
    # 请求配置
    url = f"{BASE_URL}/v1/images/generations"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # 测试用例
    test_cases = [
        {
            "name": "基础测试 - 1024x1024",
            "data": {
                "model": MODEL,
                "prompt": "a cute golden cat sitting on a chair",
                "n": 1,
                "size": "1024x1024"
            }
        },
        {
            "name": "自定义尺寸 - 1024x960",
            "data": {
                "model": MODEL,
                "prompt": "一只可爱的金色小猫坐在椅子上，阳光明媚",
                "n": 1,
                "size": "1024x960"
            }
        },
        {
            "name": "宽屏尺寸 - 1280x768",
            "data": {
                "model": MODEL,
                "prompt": "A beautiful sunset over mountains, with a lake in the foreground, photorealistic, 4k quality",
                "n": 1,
                "size": "1280x768"
            }
        },
        {
            "name": "竖屏尺寸 - 768x1280",
            "data": {
                "model": MODEL,
                "prompt": "A tall skyscraper reaching into the clouds, modern architecture",
                "n": 1,
                "size": "768x1280"
            }
        }
    ]
    
    # 执行测试
    for i, test_case in enumerate(test_cases, 1):
        print(f"\n[测试 {i}/{len(test_cases)}] {test_case['name']}")
        print("-" * 60)
        print(f"提示词: {test_case['data']['prompt']}")
        
        try:
            # 发送请求（带重试）
            print("发送请求...")
            start_time = time.time()
            
            max_retries = 3
            for retry in range(max_retries):
                try:
                    response = requests.post(
                        url,
                        headers=headers,
                        json=test_case['data'],
                        timeout=300  # 5分钟超时（因为是异步轮询）
                    )
                    break  # 成功则跳出重试循环
                except requests.exceptions.SSLError as ssl_err:
                    if retry < max_retries - 1:
                        print(f"⚠️  SSL 错误，{3} 秒后重试 ({retry + 1}/{max_retries})...")
                        time.sleep(3)
                    else:
                        raise  # 最后一次重试失败则抛出异常
            
            elapsed_time = time.time() - start_time
            
            # 检查响应
            print(f"响应状态码: {response.status_code}")
            print(f"耗时: {elapsed_time:.2f} 秒")
            
            if response.status_code == 200:
                result = response.json()
                print("✅ 请求成功！")
                
                # 显示结果
                if "data" in result and len(result["data"]) > 0:
                    print(f"生成图片数量: {len(result['data'])}")
                    for idx, img_data in enumerate(result["data"], 1):
                        if "url" in img_data:
                            print(f"  图片 {idx} URL: {img_data['url']}")
                        elif "b64_json" in img_data:
                            print(f"  图片 {idx}: Base64 编码 (长度: {len(img_data['b64_json'])} 字符)")
                    print(f"✅ 成功生成 {test_case['data']['size']} 尺寸的图片！")
                else:
                    print("⚠️  响应中没有图片数据")
                    print(f"完整响应: {json.dumps(result, indent=2, ensure_ascii=False)}")
            else:
                print(f"❌ 请求失败！")
                print(f"错误信息: {response.text}")
                
        except requests.exceptions.SSLError as ssl_err:
            print(f"❌ SSL 连接错误: {str(ssl_err)}")
            print("💡 提示: 这通常是网络问题，请稍后重试或检查网络连接")
        except requests.exceptions.Timeout:
            print("❌ 请求超时（超过5分钟）")
        except requests.exceptions.RequestException as e:
            print(f"❌ 请求异常: {str(e)}")
        except json.JSONDecodeError:
            print(f"❌ 响应解析失败: {response.text}")
        except Exception as e:
            print(f"❌ 未知错误: {str(e)}")
        
        # 测试间隔
        if i < len(test_cases):
            print("\n等待 5 秒后进行下一个测试...")
            time.sleep(5)
    
    print("\n" + "=" * 60)
    print("测试完成！")
    print("=" * 60)


def test_with_lora():
    """测试 LoRA 模型（可选）"""
    
    print("\n" + "=" * 60)
    print("LoRA 模型测试（高级功能）")
    print("=" * 60)
    
    url = f"{BASE_URL}/v1/images/generations"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    # 单个 LoRA 测试
    data = {
        "model": MODEL,
        "prompt": "a beautiful anime girl",
        "n": 1,
        "size": "1024x1024",
        "loras": "your-lora-repo-id"  # 替换为实际的 LoRA 仓库 ID
    }
    
    print("注意: 这个测试需要有效的 LoRA 仓库 ID")
    print("如果你没有 LoRA 模型，可以跳过这个测试")
    print(f"提示词: {data['prompt']}")
    
    try:
        response = requests.post(url, headers=headers, json=data, timeout=300)
        
        if response.status_code == 200:
            result = response.json()
            print("✅ LoRA 测试成功！")
            if "data" in result and len(result["data"]) > 0:
                print(f"图片 URL: {result['data'][0].get('url', 'N/A')}")
        else:
            print(f"⚠️  LoRA 测试失败: {response.text}")
            
    except Exception as e:
        print(f"⚠️  LoRA 测试异常: {str(e)}")


def quick_test():
    """快速测试 - 只测试一个简单的请求"""
    
    print("=" * 60)
    print("快速测试")
    print("=" * 60)
    
    url = f"{BASE_URL}/v1/images/generations"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    data = {
        "model": MODEL,
        "prompt": "a cute cat",
        "n": 1,
        "size": "1024x1024"
    }
    
    print(f"发送请求到: {url}")
    print(f"提示词: {data['prompt']}")
    
    try:
        start_time = time.time()
        response = requests.post(url, headers=headers, json=data, timeout=300)
        elapsed_time = time.time() - start_time
        
        print(f"响应状态码: {response.status_code}")
        print(f"耗时: {elapsed_time:.2f} 秒")
        
        if response.status_code == 200:
            result = response.json()
            print("✅ 测试成功！")
            print(f"完整响应:\n{json.dumps(result, indent=2, ensure_ascii=False)}")
        else:
            print(f"❌ 测试失败！")
            print(f"错误响应: {response.text}")
            
    except Exception as e:
        print(f"❌ 测试异常: {str(e)}")


if __name__ == "__main__":
    # 检查 API Key
    if API_KEY == "YOUR_API_KEY_HERE":
        print("❌ 错误: 请先在脚本中设置你的 API_KEY！")
        print("在脚本顶部找到 API_KEY = 'YOUR_API_KEY_HERE' 并替换为你的实际 API Key")
        exit(1)
    
    # 选择测试模式
    print("请选择测试模式:")
    print("1. 快速测试（推荐，只测试一个请求）")
    print("2. 完整测试（测试多个场景）")
    print("3. LoRA 测试（高级功能）")
    
    try:
        choice = input("\n请输入选项 (1/2/3，默认为1): ").strip() or "1"
        
        if choice == "1":
            quick_test()
        elif choice == "2":
            test_image_generation()
        elif choice == "3":
            test_with_lora()
        else:
            print("无效的选项，执行快速测试...")
            quick_test()
            
    except KeyboardInterrupt:
        print("\n\n测试已取消")
    except Exception as e:
        print(f"\n❌ 程序异常: {str(e)}")
