package main

import (
	"fmt"
	"github.com/nfnt/resize"
	"gocv.io/x/gocv"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Please input imageFile\n")
		return
	}
	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	// decode jpeg into image.Image
	rimg, err := jpeg.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(1000, 0, rimg, resize.Lanczos3)

	out, err := os.Create("test_resized.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// write new image to file
	jpeg.Encode(out, m, nil)
	img := gocv.IMRead("test_resized.jpg", gocv.IMReadColor)
	// 创建一个空的opencv mat 用于保存灰度图
	gray := gocv.NewMat()
	defer gray.Close()
	// 转化图像为灰度图
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)
	// 形态变换的预处理，得到可以查找矩形的图片
	dilation := preprocess(gray)
	defer dilation.Close()
	// 查找和筛选文字区域
	rects := findTextRegion(dilation)
	// 用绿线画出这些找到的轮廓
	for _, rect := range rects {
		tmpRects := make([][]image.Point, 0)
		tmpRects = append(tmpRects, rect)
		gocv.DrawContours(&img, gocv.NewPointsVectorFromPoints(tmpRects), 0, color.RGBA{0, 255, 0, 255}, 2)
	}
	// 5.显示带轮廓的图像
	gocv.IMWrite("imgDrawRect.jpg", img)
}

func preprocess(gray gocv.Mat) gocv.Mat {
	// Sobel算子，x方向求梯度
	sobel := gocv.NewMat()
	defer sobel.Close()
	gocv.Sobel(gray, &sobel, gocv.MatTypeCV8U, 1, 0, 3, 1, 0, gocv.BorderDefault)
	// 二值化
	binary := gocv.NewMat()
	defer binary.Close()
	gocv.Threshold(sobel, &binary, 0, 255, gocv.ThresholdOtsu+gocv.ThresholdBinary)
	// 膨胀和腐蚀操作的核函数
	element1 := gocv.GetStructuringElement(gocv.MorphRect, image.Point{30, 9})
	element2 := gocv.GetStructuringElement(gocv.MorphRect, image.Point{24, 4})
	// 膨胀，让轮廓突出
	dilation := gocv.NewMat()
	defer dilation.Close()
	gocv.Dilate(binary, &dilation, element2)
	// 腐蚀，去掉细节，如表格线等。注意这里去掉的是竖直的线
	erosion := gocv.NewMat()
	defer erosion.Close()
	gocv.Erode(dilation, &erosion, element1)
	// 再次膨胀，使轮廓明显
	dilation2 := gocv.NewMat()
	// defer dilation2.Close()
	gocv.Dilate(erosion, &dilation2, element2)
	// 存储中间图片
	gocv.IMWrite("binary.png", binary)
	gocv.IMWrite("dilation.png", dilation)
	gocv.IMWrite("erosion.png", erosion)
	gocv.IMWrite("dilation2.png", dilation2)
	return dilation2
}

func findTextRegion(img gocv.Mat) [][]image.Point {
	// 查找轮廓
	rects := make([][]image.Point, 0)
	contours := gocv.FindContours(img, gocv.RetrievalTree, gocv.ChainApproxSimple)
	for i := 0; i < contours.Size(); i++ {
		cnt := contours.At(i)
		// 计算该轮廓的面积
		area := gocv.ContourArea(cnt)
		// 面积小的都筛选掉
		// 可以调节 1000
		if area < 700 {
			continue
		}
		// 轮廓近似，作用很小
		epsilon := 0.001 * gocv.ArcLength(cnt, true)
		// approx := gocv.ApproxPolyDP(cnt, epsilon, true)
		_ = gocv.ApproxPolyDP(cnt, epsilon, true)
		// 找到最小矩形，该矩形可能有方向
		rect := gocv.MinAreaRect(cnt)
		// 计算高和宽
		mWidth := float64(rect.BoundingRect.Max.X - rect.BoundingRect.Min.X)
		mHeight := float64(rect.BoundingRect.Max.Y - rect.BoundingRect.Min.Y)
		// 筛选那些太细的矩形，留下扁的
		// 可以调节 mHeight > (mWidth * 1.2)
		if mHeight > (mWidth * 0.9) {
			continue
		}
		// 符合条件的rect添加到rects集合中
		rects = append(rects, rect.Points)
	}
	return rects
}
